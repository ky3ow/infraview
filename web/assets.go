// Package web provides static assets (templates and files) to the server.
// Build with -tags embed_assets to embed everything into the binary.
// Without the tag, assets are read from the filesystem at runtime.
package web

import (
	"html/template"
	"net/http"
)

type Assets struct {
	Login   *template.Template
	Catalog *template.Template
	Graph   *template.Template
	Static  http.FileSystem
}

var (
	loginPaths   = []string{"templates/login.html"}
	catalogPaths = []string{
		"templates/partials/layout.html",
		"templates/partials/header.html",
		"templates/catalog.html",
		"templates/partials/catalog_namespaces.html",
	}
	graphPaths = []string{
		"templates/partials/layout.html",
		"templates/partials/header.html",
		"templates/graph.html",
	}
)
