package ui

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Mux returns a HTTP Handler that knows how to serve the UI assets,
// including Javascript, HTML, and other items.
func Mux() http.Handler {
	targetURL, err := url.Parse("https://localhost:3000")
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	mux := http.NewServeMux()
	mux.Handle("/openapi/", http.StripPrefix("/openapi/", http.FileServer(http.Dir("/ui/openapi"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		headers := map[string]string{
			// Avoid page contents from being cached in either browsers or proxies.
			// This should not impact the caching of static content delivered from
			// /static routes.
			"Cache-control": "no-store, no-cache",
			// Used in pair with X-Frame-Options for the frame-ancestors part.
			// Prevent the UI from being displayed in frames from foreign domains
			// and thus avoid clickJacking.
			"Content-Security-Policy": "frame-ancestors 'self'",
			// Force use of HTTPS and prevent future uses of unencrypted HTTP
			// as protection against Man in the middle attacks.
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			// Tell browsers to follow MIME types advertised in Content-Type headers
			// and not guess them (protect against cross-site scripting and clickJacking).
			"X-Content-Type-Options": "nosniff",
			// Used in pair with Content-Security-Policy (frame-ancestors).
			// Prevent the UI from being displayed in frames from foreign domains
			// and thus avoid clickJacking.
			"X-Frame-Options": "sameorigin",
			// Protect old browsers against cross-site-scripting attacks.
			"X-XSS-Protection": "1; mode=block",
		}
		for key, value := range headers {
			w.Header().Set(key, value)
		}
		proxy.ServeHTTP(w, r)
	})
	return mux
}
