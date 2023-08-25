package ui

import (
	"net/http"
)

// Mux returns a HTTP Handler that knows how to serve the UI assets,
// including Javascript, HTML, and other items.
func Mux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/ui/static"))))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/ui/favicon.ico")
	})
	mux.HandleFunc("/service-worker.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/ui/service-worker.js")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "sameorigin")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'self'")
		http.ServeFile(w, r, "/ui/index.html")
	})
	return mux
}
