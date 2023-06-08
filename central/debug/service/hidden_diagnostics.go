package service

import (
	"net/http"

	"github.com/stackrox/rox/pkg/sac"
)

const (
	diagnosticsURL = "/diagnostics"
)

// HTTPServer is a HTTP server for exporting Prometheus metrics.
type HTTPServer struct {
	Address      string
	debugService Service
}

// NewDefaultHTTPServer creates and returns a new metrics http server with configured settings.
func NewDefaultHTTPServer(debugService Service) *HTTPServer {
	return &HTTPServer{
		Address:      ":9095",
		debugService: debugService,
	}
}

// RunForever starts the HTTP server in the background.
func (s *HTTPServer) RunForever() {
	if s == nil {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc(diagnosticsURL, func(writer http.ResponseWriter, request *http.Request) {
		ctx := sac.WithGlobalAccessScopeChecker(request.Context(), sac.AllowAllAccessScopeChecker())
		requestWithContext := request.WithContext(ctx)
		s.debugService.GetDiagnosticDumpWithCentral(writer, requestWithContext, true)
	})
	httpServer := &http.Server{
		Addr:    s.Address,
		Handler: mux,
	}
	go runForever(httpServer)
}

func runForever(server *http.Server) {
	err := server.ListenAndServe()
	// The HTTP server should never terminate.
	log.Panicf("Unexpected termination of metrics HTTP server: %v", err)
}
