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
	addr   string
	mu     sync.RWMutex
	states map[string]*graph.Graph
}

func New(addr string) *Server {
	return &Server{
		addr:   addr,
		states: make(map[string]*graph.Graph),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/api/v1/states", s.handlePlans)

	log.Printf("Listening on %s\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	path, _ := filepath.Abs("web/templates/index.html")
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	graphData := s.states["latest"]

	var encoded []byte
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
		Title string
		Graph template.JS
	}{
		Title: "Terraview",
		Graph: template.JS(encoded),
	}

	if err := tmpl.Execute(w, data); err != nil {
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
