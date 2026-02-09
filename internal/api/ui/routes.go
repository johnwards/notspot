package ui

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/johnwards/hubspot/web"
)

// RegisterRoutes registers the web UI handler at /_ui/.
func RegisterRoutes(mux *http.ServeMux) {
	distFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		panic("failed to create sub filesystem: " + err.Error())
	}

	fileServer := http.StripPrefix("/_ui/", http.FileServer(http.FS(distFS)))

	mux.HandleFunc("/_ui/", func(w http.ResponseWriter, r *http.Request) {
		// Strip the /_ui/ prefix to get the file path
		path := strings.TrimPrefix(r.URL.Path, "/_ui/")

		// Try to open the file directly
		if path != "" {
			f, err := distFS.Open(path)
			if err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// File not found or root — serve index.html for SPA routing
		indexBytes, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexBytes)
	})
}
