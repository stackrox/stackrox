package ui

import (
	"fmt"
	"net/http"

	"github.com/stackrox/rox/pkg/version"
)

// Mux returns a HTTP Handler that knows how to serve the UI assets,
// including Javascript, HTML, and other items.
func Mux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/docs/product/", http.StripPrefix("/docs/product/",
		http.RedirectHandler(fmt.Sprintf("https://docs.openshift.com/acs/%s/welcome/index.html",
			version.GetMajorMinor(version.GetMainVersion())),
			http.StatusMovedPermanently)))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/ui/static"))))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/ui/favicon.ico")
	})
	mux.HandleFunc("/service-worker.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/ui/service-worker.js")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/ui/index.html")
	})
	return mux
}
