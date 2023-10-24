package docs

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// Swagger returns an HTTP handler that exposes the swagger.json doc directly.
// It's not a gRPC method because some clients will want to consume this URL directly,
// rather than interpreting a JSON string from inside a response.
func Swagger() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, err := swaggerForRequest(req, "/stackrox/static-data/docs/api/v1/swagger.json")
		if err != nil {
			w.WriteHeader(500)
			msg := err.Error()
			_, _ = w.Write([]byte(msg))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(b)
	})
}

// SwaggerV2 returns an HTTP handler that exposes the v2 API's swagger.json doc directly.
// It's not a gRPC method because some clients will want to consume this URL directly,
// rather than interpreting a JSON string from inside a response.
func SwaggerV2() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, err := swaggerForRequest(req, "/stackrox/static-data/docs/api/v2/swagger.json")
		if err != nil {
			w.WriteHeader(500)
			msg := err.Error()
			_, _ = w.Write([]byte(msg))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(b)
	})
}

func swaggerForRequest(req *http.Request, swaggerPath string) ([]byte, error) {
	b, err := os.ReadFile(swaggerPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not load swagger file")
	}

	var swaggerSpec map[string]json.RawMessage
	if err := json.Unmarshal(b, &swaggerSpec); err != nil {
		return nil, errors.Wrap(err, "could not parse swagger spec")
	}

	swaggerSpecOut := make(map[string]interface{}, len(swaggerSpec)+2)
	for k, v := range swaggerSpec {
		swaggerSpecOut[k] = v
	}

	scheme, host := extractSchemeAndHost(req)
	swaggerSpecOut["host"] = host
	swaggerSpecOut["schemes"] = []string{scheme}

	out, err := json.MarshalIndent(swaggerSpecOut, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal swagger spec")
	}
	return out, nil
}

func extractSchemeAndHost(req *http.Request) (string, string) {
	forwardedProto := req.Header.Get("X-Forwarded-Proto")
	forwardedHost := req.Header.Get("X-Forwarded-Host")
	if forwardedHost != "" && forwardedProto != "" {
		return strings.ToLower(forwardedProto), forwardedHost
	}

	scheme := req.URL.Scheme
	if scheme == "" {
		if req.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	return scheme, host
}
