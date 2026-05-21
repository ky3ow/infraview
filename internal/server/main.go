package server

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ky3ow/infraview/pkg/graph"
)

type Server struct {
	addr        string
	catalogTmpl *template.Template
	graphTmpl   *template.Template
}

func absPaths(paths []string) []string {
	abs := make([]string, len(paths))
	for i, p := range paths {
		abs[i], _ = filepath.Abs(p)
	}
	return abs
}

func New(addr string) *Server {
	s := &Server{
		addr: addr,
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
		"web/templates/partials/catalog_namespace.html",
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
	mux.HandleFunc("GET /graph/{namespace}/{revision}", s.handleRevisionGraph)
	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("POST /api/v1/states", s.handlePlans)

	log.Printf("Listening on %s\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	type revisionChangeOverview struct {
		Created int
		Updated int
		Deleted int
	}

	type catalogRevision struct {
		SHA     string
		Branch  string
		Status  string
		Changes revisionChangeOverview
	}

	type catalogNamespace struct {
		Name      string
		Revisions []catalogRevision
	}

	data := struct {
		Title      string
		ActivePage string
		Namespaces []catalogNamespace
	}{
		Title:      "Infraview - Catalog",
		ActivePage: "catalog",
		Namespaces: []catalogNamespace{
			{
				Name: "production",
				Revisions: []catalogRevision{
					{
						SHA:    "8f4c2a91",
						Branch: "simple_deps",
						Status: "applied",
						Changes: revisionChangeOverview{
							Created: 3,
							Updated: 1,
							Deleted: 0,
						},
					},
				},
			},
			{
				Name: "loops",
				Revisions: []catalogRevision{
					{
						SHA:    "1",
						Branch: "12-create-count",
						Status: "applied",
						Changes: revisionChangeOverview{
							Created: 0,
							Updated: 0,
							Deleted: 0,
						},
					},
					{
						SHA:    "2",
						Branch: "13-create-foreach",
						Status: "applied",
						Changes: revisionChangeOverview{
							Created: 0,
							Updated: 0,
							Deleted: 0,
						},
					},
					{
						SHA:    "3",
						Branch: "14-delete-count",
						Status: "applied",
						Changes: revisionChangeOverview{
							Created: 0,
							Updated: 0,
							Deleted: 0,
						},
					},
				},
			},
			{
				Name: "modules",
				Revisions: []catalogRevision{
					{
						SHA: "05",
						Branch: "05-module-vars",
						Status: "applied",
						Changes: revisionChangeOverview{},
					},
					{
						SHA: "07",
						Branch: "07-module-outputs",
						Status: "applied",
						Changes: revisionChangeOverview{},
					},
					{
						SHA: "08",
						Branch: "08-complex-vars",
						Status: "applied",
						Changes: revisionChangeOverview{},
					},
					{
						SHA: "10",
						Branch: "10-module-deletions",
						Status: "applied",
						Changes: revisionChangeOverview{},
					},
				},
			},
			{
				Name: "staging",
				Revisions: []catalogRevision{
					{
						SHA:    "c71b03de",
						Branch: "feature/db-resize",
						Status: "pending",
						Changes: revisionChangeOverview{
							Created: 1,
							Updated: 4,
							Deleted: 0,
						},
					},
					{
						SHA:    "1a9e6b42",
						Status: "failed",
						Changes: revisionChangeOverview{
							Created: 0,
							Updated: 2,
							Deleted: 5,
						},
					},
				},
			},
		},
	}

	if err := s.catalogTmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	// TODO redirect to /graph/{latest_namespace}/{latest_revision} once catalog metadata is stored.
	http.Error(w, "latest graph is not available yet", http.StatusNotImplemented)
}

func (s *Server) handleRevisionGraph(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	revision := r.PathValue("revision")

	graphData, err := getGraph(namespace, revision)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title      string
		ActivePage string
		Graph      graph.Graph
	}{
		Title:      "Infraview - Graph",
		ActivePage: "graph",
		Graph:      *graphData,
	}

	if err := s.graphTmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getGraph(namespace, revision string) (*graph.Graph, error) {
	// TODO replace this shim with a namespace/revision lookup backed by persisted plans.
	_ = namespace

	plans := map[string]string{
		"8f4c2a91": "test/graph/03-simple-dependencies/plan.json",
		"c71b03de": "test/graph/02-simple-create-update-delete/plan.json",
		"1a9e6b42": "test/graph/11-deletion-edges/plan.json",
		"1": "test/graph/12-create-count/plan.json",
		"2": "test/graph/13-create-foreach/plan.json",
		"3": "test/graph/14-delete-count/plan.json",
		"05": "test/graph/05-module-vars/plan.json",
		"07": "test/graph/07-module-outputs/plan.json",
		"08": "test/graph/08-complex-vars/plan.json",
		"10": "test/graph/10-module-deletions/plan.json",
	}

	planPath, ok := plans[revision]
	if !ok {
		return nil, os.ErrNotExist
	}

	file, err := os.Open(planPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return graph.New(file)
}

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	// TODO store the submitted plan in DB with namespace/revision metadata.
	g, err := graph.New(r.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.Marshal(g)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error parsing plan %v", err)
		return
	}

	_, err = w.Write(jsonData)

	if err != nil {
		log.Printf("Error writing plan response %v", err)
	}
}
