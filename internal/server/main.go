package server

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/ky3ow/infraview/pkg/auth"
	"github.com/ky3ow/infraview/pkg/catalog"
	"github.com/ky3ow/infraview/pkg/graph"
	"github.com/ky3ow/infraview/pkg/storage"
)

type Server struct {
	addr               string
	catalogTmpl        *template.Template
	graphTmpl          *template.Template
	store              *storage.Store
}

func absPaths(paths []string) []string {
	abs := make([]string, len(paths))
	for i, p := range paths {
		abs[i], _ = filepath.Abs(p)
	}
	return abs
}

func New(addr string, store *storage.Store) *Server {
	s := &Server{
		addr:  addr,
		store: store,
	}
	s.loadTemplates()
	return s
}

func (s *Server) loadTemplates() {
	base := absPaths([]string{
		"web/templates/partials/layout.html",
		"web/templates/partials/header.html",
	})

	catalog := append([]string{}, base...)
	catalog = append(catalog, absPaths([]string{
		"web/templates/catalog.html",
		"web/templates/partials/catalog_namespaces.html",
	})...)

	graph := append([]string{}, base...)
	graph = append(graph, absPaths([]string{"web/templates/graph.html"})...)

	s.catalogTmpl = template.Must(template.ParseFiles(catalog...))
	s.graphTmpl = template.Must(template.ParseFiles(graph...))
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleCatalog)
	mux.HandleFunc("GET /graph", s.handleGraph)
	mux.HandleFunc("GET /graph/{namespace}", s.handleGraphNamespace)
	mux.HandleFunc("GET /namespaces/{namespace}/revisions", s.handleGraphRevisions)
	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("GET /graph/{namespace}/{revision}", s.handleRevisionGraph)
	mux.HandleFunc("PUT /api/v1/graph/{namespace}/{revision}", s.withAuth(auth.Write, s.handlePutGraph))
	mux.HandleFunc("GET /api/catalog/search", s.handleCatalogSearch)

	log.Printf("Listening on %s\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// withAuth extracts a Bearer token from the Authorization header,
// hashes it, and verifies a matching token of the required type exists.
func (s *Server) withAuth(requiredType auth.Type, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			http.Error(w, "unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		raw := strings.TrimPrefix(header, "Bearer ")
		if raw == header {
			http.Error(w, "unauthorized: expected Bearer token", http.StatusUnauthorized)
			return
		}

		t := auth.FromRaw(raw, requiredType)
		_, err := s.store.FindTokenByHash(t.Hash())
		if err != nil {
			http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	namespaces, err := catalog.SearchCatalog(s.store, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title      string
		ActivePage string
		Namespaces []catalog.CatalogResult
		Query      string
	}{
		Title:      "Infraview - Catalog",
		ActivePage: "catalog",
		Namespaces: namespaces,
		Query:      query,
	}

	renderTarget := "layout"
	if r.Header.Get("HX-Request") == "true" {
		renderTarget = "catalog-namespaces"
	}
	if err := s.catalogTmpl.ExecuteTemplate(w, renderTarget, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	lr, err := catalog.GetLatestRevision(s.store)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/graph/"+lr.NamespaceName+"/"+lr.Hash, http.StatusSeeOther)
}

func (s *Server) handleGraphNamespace(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	lr, err := catalog.GetLatestRevisionByNamespace(s.store, namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/graph/"+lr.NamespaceName+"/"+lr.Hash, http.StatusSeeOther)
}

func (s *Server) handleGraphRevisions(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	if namespace == "" {
		return
	}

	revisions, err := catalog.ListRevisionsByName(s.store, namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Revisions  []catalog.RevisionView
		SelectedRevision string
	}{
		Revisions: revisions,
		SelectedRevision: "",
	}

	if err := s.graphTmpl.ExecuteTemplate(w, "revision-options", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleRevisionGraph(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	revision := r.PathValue("revision")

	graphData, err := catalog.GetRevisionGraph(s.store, namespace, revision)
	if errors.Is(err, storage.ErrRevisionNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nsList, err := catalog.ListNamespaces(s.store)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	revisions, err := catalog.ListRevisionsByName(s.store, namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title             string
		ActivePage        string
		Graph             graph.Graph
		Namespaces        []catalog.NamespaceView
		Revisions         []catalog.RevisionView
		SelectedNamespace string
		SelectedRevision  string
	}{
		Title:             "Infraview - Graph",
		ActivePage:        "graph",
		Graph:             *graphData,
		Namespaces:        nsList,
		Revisions:         revisions,
		SelectedNamespace: namespace,
		SelectedRevision:  revision,
	}

	renderTarget := "layout"
	if r.Header.Get("HX-Request") == "true" {
		renderTarget = "content"
		w.Header().Set("HX-Push-Url", "/graph/"+namespace+"/"+revision)
	}
	if err := s.graphTmpl.ExecuteTemplate(w, renderTarget, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handlePutGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.PathValue("namespace")
	revision := r.PathValue("revision")

	var name *string
	if n := r.URL.Query().Get("name"); n != "" {
		name = &n
	}

	var plan tfjson.Plan
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&plan); err != nil {
		http.Error(w, "invalid terraform plan json: "+err.Error(), http.StatusBadRequest)
		return
	}

	g, err := graph.New(&plan)
	if err != nil {
		http.Error(w, "failed to build graph: "+err.Error(), http.StatusBadRequest)
		return
	}

	metadata := catalog.ExtractPlanMetadata(plan)

	rev, err := catalog.SavePlan(s.store, namespace, revision, name, *metadata, g)
	if err != nil {
		http.Error(w, "failed to save plan", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rev)
}

func (s *Server) handleCatalogSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	results, err := catalog.SearchCatalog(s.store, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if results == nil {
		results = []catalog.CatalogResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
