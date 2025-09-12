package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/stackrox/rox/central/centralproxy/pkg/auth"
)

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Infof("%s %s %d %v", r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

// authenticationMiddleware validates OpenShift tokens and extracts user info
func (s *Server) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract Bearer token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Warn("Missing Authorization header")
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("Invalid Authorization header format")
			http.Error(w, "Bearer token required", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			log.Warn("Empty bearer token")
			http.Error(w, "Valid bearer token required", http.StatusUnauthorized)
			return
		}

		// Validate token and get user info
		userInfo, err := s.authValidator.ValidateToken(r.Context(), token)
		if err != nil {
			log.Errorf("Token validation failed: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user info to request context (explicitly cast to ensure auth import is used)
		ctx := context.WithValue(r.Context(), "userInfo", (*auth.UserInfo)(userInfo))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}