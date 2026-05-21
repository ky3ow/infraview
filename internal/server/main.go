package server

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/ky3ow/infraview/pkg/graph"
)

type Server struct {
	addr        string
	mu          sync.RWMutex
	states      map[string]*graph.Graph
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
		addr:   addr,
		states: make(map[string]*graph.Graph),
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
	mux.HandleFunc("/", s.handleCatalog)
	mux.HandleFunc("/graph", s.handleGraph)
	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/api/v1/states", s.handlePlans)

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
						Branch: "main",
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
	s.mu.Lock()
	defer s.mu.Unlock()
	graphData := s.states["latest"]

	var encoded []byte
	var err error
	if graphData != nil {
		encoded, err = json.Marshal(graphData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		encoded = []byte(`{"nodes":[],"edges":[]}`)
	}

	data := struct {
		Title      string
		ActivePage string
		Graph      template.JS
	}{
		Title:      "Infraview - Graph",
		ActivePage: "graph",
		Graph:      template.JS(encoded),
	}

	if err := s.graphTmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	g, err := graph.New(r.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.states["latest"] = g

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
