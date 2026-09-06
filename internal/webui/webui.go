// Package webui embeds and serves the AISwitcher web app (Vite + React 19 +
// shadcn/ui) built in web/ and synced into dist/ by `task web:build`.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// distFiles holds the built SPA. The checked-in placeholder keeps the embed
// valid before the first frontend build.
//
//go:embed all:dist
var distFiles embed.FS

// Handler returns an http.Handler serving the embedded SPA with client-side
// routing fallback: unknown paths return index.html so TanStack Router can
// resolve deep links like /terminal.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(distFiles, "dist")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if _, err := fs.Stat(sub, path); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// SPA fallback: rewrite to the shell document.
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	}), nil
}
