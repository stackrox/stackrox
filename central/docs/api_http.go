package docs

import (
	"io/ioutil"
	"net/http"
)

// Swagger returns an HTTP handler that exposes the swagger.json doc directly.
// It's not a gRPC method because some clients will want to consume this URL directly,
// rather than interpreting a JSON string from inside a response.
func Swagger() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, err := ioutil.ReadFile("/docs/api/v1/swagger.json")
		if err != nil {
			w.WriteHeader(500)
			msg := err.Error()
			w.Write([]byte(msg))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(b)
	})
}
