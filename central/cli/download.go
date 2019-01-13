package cli

import (
	"net/http"
	"path/filepath"
)

const downloadPath = "/assets/downloads/cli"

// Handler returns a handler for serving files from Central's downloads folder
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Need to remove the rest of the URL path so that it just contains the file wanted
		r.URL.Path = filepath.Base(r.URL.Path)
		http.FileServer(http.Dir(downloadPath)).ServeHTTP(w, r)
	}
}
