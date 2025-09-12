package server

import (
	"io"
	"net/http"

	"github.com/stackrox/rox/central/centralproxy/pkg/auth"
	"github.com/stackrox/rox/central/centralproxy/pkg/permissions"
	"github.com/stackrox/rox/central/centralproxy/pkg/proxy"
	"github.com/stackrox/rox/central/centralproxy/pkg/rbac"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()
)

// Config holds the configuration for the Central Proxy server
type Config struct {
	Port          string
	K8sClient     kubernetes.Interface
	CentralClient interface{} // TODO: Replace with actual Central client type
	Namespace     string
}

// Server represents the Central Proxy HTTP server
type Server struct {
	config              *Config
	mux                 *http.ServeMux
	authValidator       *auth.Validator
	rbacChecker         *rbac.Checker
	proxyHandler        *proxy.Handler
	permissionsAnalyzer *permissions.GraphQLQueryAnalyzer
}

// New creates a new Central Proxy server
func New(config *Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
	}

	// Initialize components
	s.authValidator = auth.NewValidator()
	s.rbacChecker = rbac.NewChecker(config.K8sClient)
	s.proxyHandler = proxy.NewHandler(config.CentralClient)
	s.permissionsAnalyzer = permissions.NewGraphQLQueryAnalyzer()

	// Setup routes
	s.setupRoutes()

	return s
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Apply middleware manually since http.ServeMux doesn't have middleware support
	handler := s.applyMiddleware(s.mux)
	handler.ServeHTTP(w, r)
}

// setupRoutes configures all HTTP routes for the proxy
func (s *Server) setupRoutes() {
	// Health check endpoints
	s.mux.HandleFunc("/health", s.methodFilter("GET", s.handleHealth))
	s.mux.HandleFunc("/ready", s.methodFilter("GET", s.handleReady))

	// GraphQL endpoint (primary for MVP)
	s.mux.HandleFunc("/graphql", s.methodFilter("POST", s.handleGraphQL))

	// Future REST API endpoints (extensible design)
	s.mux.HandleFunc("/api/v1/images", s.methodFilter("GET", s.handleImages))
	s.mux.HandleFunc("/api/v1/vulnerabilities", s.methodFilter("GET", s.handleVulnerabilities))
	s.mux.HandleFunc("/api/v1/policies", s.methodFilter("GET", s.handlePolicies))
}

// methodFilter wraps a handler to only accept specific HTTP methods
func (s *Server) methodFilter(allowedMethod string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != allowedMethod {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}

// applyMiddleware wraps the handler with middleware
func (s *Server) applyMiddleware(handler http.Handler) http.Handler {
	// Apply in reverse order since each middleware wraps the next
	handler = s.authenticationMiddleware(handler)
	handler = s.loggingMiddleware(handler)
	return handler
}

// Health check handlers
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// TODO: Add readiness checks (Central connectivity, etc.)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

// GraphQL handler (MVP implementation)
func (s *Server) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	log.Debug("Handling GraphQL request")

	// Get user info from authentication middleware
	userInfo, ok := r.Context().Value("userInfo").(*auth.UserInfo)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse GraphQL query and determine required permissions
	requiredPermissions, err := s.parseGraphQLPermissions(r)
	if err != nil {
		log.Errorf("Failed to parse GraphQL permissions: %v", err)
		http.Error(w, "Invalid GraphQL query", http.StatusBadRequest)
		return
	}

	// Check permissions
	for _, perm := range requiredPermissions {
		if !s.rbacChecker.CheckAccess(r.Context(), userInfo, perm.Resource, perm.Verb) {
			log.Warnf("User %s denied access to %s:%s", userInfo.Username, perm.Resource, perm.Verb)
			http.Error(w, "Insufficient permissions", http.StatusForbidden)
			return
		}
	}

	// Note: parseGraphQLPermissions has consumed the request body,
	// so we need to create a new request for the proxy handler
	// In a production implementation, we should avoid reading the body twice
	s.proxyHandler.HandleGraphQL(w, r, userInfo)
}

// Future REST API handlers (extensible design)
func (s *Server) handleImages(w http.ResponseWriter, r *http.Request) {
	userInfo := r.Context().Value("userInfo").(*auth.UserInfo)
	
	if !s.rbacChecker.CheckAccess(r.Context(), userInfo, "images", "list") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	s.proxyHandler.HandleImages(w, r, userInfo)
}

func (s *Server) handleVulnerabilities(w http.ResponseWriter, r *http.Request) {
	userInfo := r.Context().Value("userInfo").(*auth.UserInfo)
	
	if !s.rbacChecker.CheckAccess(r.Context(), userInfo, "vulnerabilities", "list") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	s.proxyHandler.HandleVulnerabilities(w, r, userInfo)
}

func (s *Server) handlePolicies(w http.ResponseWriter, r *http.Request) {
	userInfo := r.Context().Value("userInfo").(*auth.UserInfo)
	
	if !s.rbacChecker.CheckAccess(r.Context(), userInfo, "policies", "list") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	s.proxyHandler.HandlePolicies(w, r, userInfo)
}

// parseGraphQLPermissions analyzes a GraphQL query to determine required permissions
func (s *Server) parseGraphQLPermissions(r *http.Request) ([]rbac.Permission, error) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Use the permissions analyzer to extract required permissions
	permissions, err := s.permissionsAnalyzer.ExtractPermissions(body)
	if err != nil {
		return nil, err
	}

	return permissions, nil
}