package server

import (
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/ky3ow/infraview/pkg/auth"
	"github.com/ky3ow/infraview/pkg/catalog"
	"github.com/ky3ow/infraview/pkg/graph"
	"github.com/ky3ow/infraview/pkg/log"
	"github.com/ky3ow/infraview/pkg/storage"
)

const sessionCookieName = "session"

var logger = log.New("server")

type Server struct {
	addr     string
	store    *storage.Store
	sessions *auth.SessionStore
	noAuth   bool

	loginTmpl   *template.Template
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

func New(addr string, store *storage.Store, noAuth bool) *Server {
	s := &Server{
		addr:     addr,
		store:    store,
		sessions: auth.NewSessionStore(),
		noAuth:   noAuth,
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
	s.loginTmpl = template.Must(template.ParseFiles(
		absPaths([]string{"web/templates/login.html"})...,
	))
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", s.handleLogin)
	mux.HandleFunc("POST /login", s.handleLoginSubmit)
	mux.HandleFunc("POST /logout", s.handleLogout)
	mux.HandleFunc("GET /{$}", s.requireSession(s.handleCatalog))
	mux.HandleFunc("GET /graph", s.requireSession(s.handleGraph))
	mux.HandleFunc("GET /graph/{namespace}", s.requireSession(s.handleGraphNamespace))
	mux.HandleFunc("GET /namespaces/{namespace}/revisions", s.requireSession(s.handleGraphRevisions))
	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("GET /graph/{namespace}/{revision}", s.requireSession(s.handleRevisionGraph))
	mux.HandleFunc("PUT /api/v1/graph/{namespace}/{revision}", s.requireApiAuth(auth.Write, s.handlePutGraph))

	logger.Info("listening", "addr", s.addr)
	return http.ListenAndServe(s.addr, s.logRequests(mux))
}

// logRequests is HTTP middleware that logs each request with method, path, status, and duration.
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			level := slog.LevelInfo
			if lw.status >= 500 {
				level = slog.LevelError
			} else if lw.status >= 400 {
				level = slog.LevelWarn
			}
			logger.Log(r.Context(), level, "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", lw.status,
				"duration", time.Since(start).String(),
			)
		}()
		next.ServeHTTP(lw, r)
	})
}

// loggingResponseWriter captures the HTTP status code.
type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.status = code
	lw.ResponseWriter.WriteHeader(code)
}

// requireApiAuth checks the Authorization: Bearer header against stored tokens.
// Returns 401 if missing or invalid. Used for programmatic API endpoints.
// Skipped entirely when noAuth is set.
func (s *Server) requireApiAuth(requiredType auth.Type, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.noAuth {
			next(w, r)
			return
		}

		header := r.Header.Get("Authorization")
		if header == "" {
			logger.Warn("api auth denied: missing token", "method", r.Method, "path", r.URL.Path)
			http.Error(w, "unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		raw := strings.TrimPrefix(header, "Bearer ")
		if raw == header {
			logger.Warn("api auth denied: invalid auth header format", "method", r.Method, "path", r.URL.Path)
			http.Error(w, "unauthorized: expected Bearer token", http.StatusUnauthorized)
			return
		}

		t := auth.FromRaw(raw, requiredType)
		_, err := s.store.FindTokenByHash(t.Hash())
		if err != nil {
			logger.Warn("api auth denied: invalid token", "method", r.Method, "path", r.URL.Path, "error", err)
			http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// requireSession checks for a valid session cookie.
// Redirects to /login if missing or expired. Used for interactive page routes.
// Skipped entirely when noAuth is set.
func (s *Server) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.noAuth {
			next(w, r)
			return
		}

		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			if s.sessions.Get(cookie.Value) != nil {
				next(w, r)
				return
			}
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.noAuth {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Already have a valid session? Redirect to catalog.
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if s.sessions.Get(cookie.Value) != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	s.loginTmpl.ExecuteTemplate(w, "layout", nil)
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	rawToken := r.FormValue("token")
	if rawToken == "" {
		s.loginTmpl.ExecuteTemplate(w, "layout", "token required")
		return
	}

	t := auth.FromRaw(rawToken, auth.Read)
	tok, err := s.store.FindTokenByHash(t.Hash())
	if err != nil {
		logger.Info("login failed: invalid token", "remote", r.RemoteAddr)
		s.loginTmpl.ExecuteTemplate(w, "layout", "invalid token")
		return
	}

	sess := s.sessions.Create(auth.Type(tok.Type))

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(auth.SessionTTL.Seconds()),
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	namespaces, err := catalog.SearchCatalog(s.store, query)
	if err != nil {
		logger.Error("catalog search failed", "error", err, "query", query)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title      string
		ActivePage string
		Namespaces []catalog.CatalogResult
		Query      string
		NoAuth     bool
	}{
		Title:      "Infraview - Catalog",
		ActivePage: "catalog",
		Namespaces: namespaces,
		Query:      query,
		NoAuth:     s.noAuth,
	}

	renderTarget := "layout"
	if r.Header.Get("HX-Request") == "true" {
		renderTarget = "catalog-namespaces"
	}
	if err := s.catalogTmpl.ExecuteTemplate(w, renderTarget, data); err != nil {
		logger.Error("failed to render catalog template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	lr, err := catalog.GetLatestRevision(s.store)
	if err != nil {
		logger.Error("failed to get latest revision", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/graph/"+lr.NamespaceName+"/"+lr.Hash, http.StatusSeeOther)
}

func (s *Server) handleGraphNamespace(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	lr, err := catalog.GetLatestRevisionByNamespace(s.store, namespace)
	if err != nil {
		logger.Error("failed to get latest revision by namespace", "error", err, "namespace", namespace)
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
		logger.Error("failed to list revisions", "error", err, "namespace", namespace)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Revisions        []catalog.RevisionView
		SelectedRevision string
	}{
		Revisions:        revisions,
		SelectedRevision: "",
	}

	if err := s.graphTmpl.ExecuteTemplate(w, "revision-options", data); err != nil {
		logger.Error("failed to render revision options", "error", err, "namespace", namespace)
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
		logger.Error("failed to get revision graph", "error", err, "namespace", namespace, "revision", revision)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nsList, err := catalog.ListNamespaces(s.store)
	if err != nil {
		logger.Error("failed to list namespaces", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	revisions, err := catalog.ListRevisionsByName(s.store, namespace)
	if err != nil {
		logger.Error("failed to list revisions for graph page", "error", err, "namespace", namespace)
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
		NoAuth            bool
	}{
		Title:             "Infraview - Graph",
		ActivePage:        "graph",
		Graph:             *graphData,
		Namespaces:        nsList,
		Revisions:         revisions,
		SelectedNamespace: namespace,
		SelectedRevision:  revision,
		NoAuth:            s.noAuth,
	}

	renderTarget := "layout"
	if r.Header.Get("HX-Request") == "true" {
		renderTarget = "content"
		w.Header().Set("HX-Push-Url", "/graph/"+namespace+"/"+revision)
	}
	if err := s.graphTmpl.ExecuteTemplate(w, renderTarget, data); err != nil {
		logger.Error("failed to render graph template", "error", err, "namespace", namespace, "revision", revision)
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
		logger.Warn("invalid plan json received", "error", err, "namespace", namespace, "revision", revision)
		http.Error(w, "invalid terraform plan json: "+err.Error(), http.StatusBadRequest)
		return
	}

	g, err := graph.New(&plan)
	if err != nil {
		logger.Warn("failed to build graph from plan", "error", err, "namespace", namespace, "revision", revision)
		http.Error(w, "failed to build graph: "+err.Error(), http.StatusBadRequest)
		return
	}

	metadata := catalog.ExtractPlanMetadata(plan)

	rev, err := catalog.SavePlan(s.store, namespace, revision, name, *metadata, g)
	if err != nil {
		logger.Error("failed to save plan", "error", err, "namespace", namespace, "revision", revision)
		http.Error(w, "failed to save plan", http.StatusInternalServerError)
		return
	}

	logger.Info("plan saved", "namespace", namespace, "revision", revision,
		"created", metadata.Created, "updated", metadata.Updated, "deleted", metadata.Deleted)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rev)
}
