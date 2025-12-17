package graphqlgateway

import (
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"google.golang.org/grpc/codes"
)

const (
	// GraphQLPath is the Central GraphQL API path
	GraphQLPath = "/api/graphql"

	// HeaderNamespace is the HTTP header for namespace scope
	HeaderNamespace = "X-Namespace"

	// HeaderDeployment is the HTTP header for deployment scope
	HeaderDeployment = "X-Deployment"

	// HeaderAuthorization is the standard Authorization header
	HeaderAuthorization = "Authorization"

	// HeaderTraceID is the trace ID header for request correlation
	HeaderTraceID = "X-Trace-ID"

	// BearerPrefix is the prefix for bearer tokens
	BearerPrefix = "Bearer "
)

var (
	log = logging.LoggerForModule()
)

// TokenManager defines the interface for acquiring scoped tokens.
// This interface allows for easier testing by enabling mock implementations.
type TokenManager interface {
	GetToken(ctx context.Context, bearerToken, namespace, deployment string) (string, error)
}

// Handler handles GraphQL requests from the OCP console plugin,
// validates Kubernetes RBAC, acquires scoped tokens, and proxies
// queries to Central's GraphQL API.
type Handler struct {
	centralClient    *http.Client
	tokenManager     TokenManager
	centralReachable atomic.Bool
}

// NewHandler creates a new GraphQL gateway handler.
//
// Parameters:
// - centralEndpoint: The Central HTTP endpoint (e.g., "https://central.stackrox:443")
// - centralCertificates: Central's CA certificates for mTLS
// - tokenManager: Token manager for acquiring scoped tokens
func NewHandler(
	centralEndpoint string,
	centralCertificates []*x509.Certificate,
	tokenManager TokenManager,
) (*Handler, error) {
	client, err := centralclient.AuthenticatedCentralHTTPClient(centralEndpoint, centralCertificates)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating central HTTP transport")
	}

	return &Handler{
		centralClient: client,
		tokenManager:  tokenManager,
	}, nil
}

// Notify reacts to Sensor going into online/offline mode.
func (h *Handler) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, "GraphQL gateway handler"))
	switch e {
	case common.SensorComponentEventCentralReachable:
		h.centralReachable.Store(true)
	case common.SensorComponentEventOfflineMode:
		h.centralReachable.Store(false)
	}
}

// ServeHTTP handles HTTP requests to the GraphQL gateway.
//
// Request flow:
// 1. Validate request (POST only, has Authorization header)
// 2. Extract namespace/deployment from headers
// 3. Extract OCP bearer token
// 4. Validate K8s RBAC and acquire scoped token (via TokenManager)
// 5. Proxy GraphQL query to Central with scoped token
// 6. Return response
func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Generate trace ID for request correlation
	traceID := uuid.NewV4().String()
	writer.Header().Set(HeaderTraceID, traceID)

	// Validate HTTP method
	if request.Method != http.MethodPost {
		log.Warnw("Invalid HTTP method for GraphQL request",
			logging.String("method", request.Method),
			logging.String("trace_id", traceID),
		)
		httputil.WriteGRPCStyleErrorf(writer, codes.InvalidArgument, "only POST requests are allowed")
		return
	}

	// Extract scope from headers
	namespace := request.Header.Get(HeaderNamespace)
	deployment := request.Header.Get(HeaderDeployment)

	// Extract OCP bearer token
	authHeader := request.Header.Get(HeaderAuthorization)
	if authHeader == "" {
		log.Warnw("Missing Authorization header",
			logging.String("trace_id", traceID),
		)
		httputil.WriteGRPCStyleErrorf(writer, codes.Unauthenticated, "missing Authorization header")
		return
	}

	if len(authHeader) <= len(BearerPrefix) || authHeader[:len(BearerPrefix)] != BearerPrefix {
		// Bearer prefix was not found
		log.Warnw("Invalid Authorization header format (missing 'Bearer ' prefix)",
			logging.String("trace_id", traceID),
		)
		httputil.WriteGRPCStyleErrorf(writer, codes.Unauthenticated, "invalid Authorization header format")
		return
	}
	ocpToken := authHeader[len(BearerPrefix):]

	// Acquire scoped token (validates K8s RBAC and gets/creates scoped token)
	scopedToken, err := h.tokenManager.GetToken(request.Context(), ocpToken, namespace, deployment)
	if err != nil {
		// TokenManager returns appropriate error types:
		// - NoCredentials: invalid token
		// - NotAuthorized: RBAC denied
		// - ServerError: Central offline or other errors
		log.Warnw("Failed to acquire scoped token",
			logging.Err(err),
			logging.String("namespace", namespace),
			logging.String("deployment", deployment),
			logging.String("trace_id", traceID),
		)

		// Determine appropriate gRPC code from error type
		code := codes.Internal
		if errox.IsAny(err, errox.NotAuthorized) {
			code = codes.PermissionDenied
		} else if errox.IsAny(err, errox.NoCredentials) {
			code = codes.Unauthenticated
		} else if errox.IsAny(err, errox.ServerError) {
			code = codes.Unavailable
		}

		httputil.WriteGRPCStyleErrorf(writer, code, "authorization failed: %v", err)
		return
	}

	// Prepare Central's GraphQL request
	centralURL := url.URL{
		Path: GraphQLPath,
	}

	centralRequest, err := http.NewRequestWithContext(
		request.Context(),
		http.MethodPost,
		centralURL.String(),
		request.Body,
	)
	if err != nil {
		log.Errorw("Failed to create Central request",
			logging.Err(err),
			logging.String("trace_id", traceID),
		)
		httputil.WriteGRPCStyleErrorf(writer, codes.Internal, "failed to create request: %v", err)
		return
	}

	// Set headers for Central request
	centralRequest.Header.Set(HeaderAuthorization, BearerPrefix+scopedToken)
	centralRequest.Header.Set("Content-Type", "application/json")
	centralRequest.Header.Set(HeaderTraceID, traceID)

	// Log the proxied request
	log.Infow("Proxying GraphQL query to Central",
		logging.String("namespace", namespace),
		logging.String("deployment", deployment),
		logging.String("trace_id", traceID),
	)

	// Execute request to Central
	resp, err := h.centralClient.Do(centralRequest)
	if err != nil {
		log.Errorw("Failed to contact Central",
			logging.Err(err),
			logging.String("trace_id", traceID),
		)
		httputil.WriteGRPCStyleErrorf(writer, codes.Internal, "failed to contact central: %v", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnw("Failed to close response body",
				logging.Err(err),
				logging.String("trace_id", traceID),
			)
		}
	}()

	// Copy response headers from Central
	for k, vs := range resp.Header {
		for _, v := range vs {
			writer.Header().Add(k, v)
		}
	}

	// Write response status and body
	writer.WriteHeader(resp.StatusCode)
	bytesWritten, err := io.Copy(writer, resp.Body)
	if err != nil {
		log.Warnw("Error copying response body",
			logging.Err(err),
			logging.String("trace_id", traceID),
		)
		return
	}

	log.Infow("GraphQL query completed",
		logging.String("namespace", namespace),
		logging.String("deployment", deployment),
		logging.Int("status_code", resp.StatusCode),
		logging.Int("response_bytes", int(bytesWritten)),
		logging.String("trace_id", traceID),
	)
}
