package ui

import (
	"net/http"

	"github.com/stackrox/rox/central/ed"
)

// Mux returns a HTTP Handler that knows how to serve the UI assets,
// including Javascript, HTML, and other items.
func Mux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/docs/product/", http.StripPrefix("/docs/product/", http.FileServer(http.Dir(ed.PED("product-docs")))))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(ed.PED("ui/static")))))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, ed.PED("ui/favicon.ico"))
	})
	mux.HandleFunc("/service-worker.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, ed.PED("ui/service-worker.js"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, ed.PED("ui/index.html"))
	})
	return mux
}

// RestrictedModeMux returns a HTTP handler that serves a static "you need a license to use this product"
// message.
func RestrictedModeMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/stackrox/ui/index.html")
	})
	return mux
}
