//go:build embed_assets

package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed static templates
var webFS embed.FS

func Load() *Assets {
	staticDir, _ := fs.Sub(webFS, "static")
	return &Assets{
		Login:   parse(loginPaths),
		Catalog: parse(catalogPaths),
		Graph:   parse(graphPaths),
		Static:  http.FS(staticDir),
	}
}

// Paths are used directly against the embedded FS rooted at web/.
func parse(paths []string) *template.Template {
	return template.Must(template.ParseFS(webFS, paths...))
}
