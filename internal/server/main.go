package server

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

// Server runs the HTTP app.
type Server struct {
	addr string
}

// New creates a new Server bound to addr.
func New(addr string) *Server {
	return &Server{addr: addr}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	log.Printf("Listening on %s\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	path, _ := filepath.Abs("web/templates/index.html")
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	data := map[string]string{
		"Title":   "Terraform Visualizer",
		"Message": "Server is running.",
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
