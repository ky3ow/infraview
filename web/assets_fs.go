//go:build !embed_assets

package web

import (
	"html/template"
	"net/http"
	"path/filepath"
)

func Load() *Assets {
	return &Assets{
		Login:   parse(loginPaths),
		Catalog: parse(catalogPaths),
		Graph:   parse(graphPaths),
		Static:  http.Dir("web/static"),
	}
}

// Each path is prefixed with "web/" and resolved to an absolute path.
func parse(paths []string) *template.Template {
	abs := make([]string, len(paths))
	for i, p := range paths {
		abs[i], _ = filepath.Abs(filepath.Join("web", p))
	}
	return template.Must(template.ParseFiles(abs...))
}
