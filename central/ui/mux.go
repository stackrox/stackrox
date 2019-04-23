package ui

import (
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/stackrox/rox/central/encdata"
)

// Mux returns a HTTP Handler that knows how to serve the UI assets,
// including Javascript, HTML, and other items.
func Mux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/docs/product/", http.StripPrefix("/docs/product/", http.FileServer(http.Dir(encdata.PrefixExtractedDir("product-docs")))))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(encdata.PrefixExtractedDir("ui/static")))))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, encdata.PrefixExtractedDir("ui/favicon.ico"))
	})
	mux.HandleFunc("/service-worker.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, encdata.PrefixExtractedDir("ui/service-worker.js"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, encdata.PrefixExtractedDir("ui/index.html"))
	})
	return gziphandler.GzipHandler(mux)
}
