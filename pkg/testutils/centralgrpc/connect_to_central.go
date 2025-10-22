package centralgrpc

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	usernameEnvVar = "ROX_USERNAME"

	apiEndpointEnvVar = "API_ENDPOINT"

	defaultUsername = "admin"
	//#nosec G101 -- This is a false positive
	defaultPasswordPath = "deploy/k8s/central-deploy/password"
	defaultAPIEndpoint  = "localhost:8000"
)

var (
	_, inCI = os.LookupEnv("CI")
)

func mustGetEnvVarInCI(t testutils.T, envVar string) string {
	value := os.Getenv(envVar)
	if inCI {
		require.NotEmpty(t, value, "Please set %s", envVar)
	}
	return value
}

// RoxUsername returns the rox basic auth username (or fails the test if not found).
func RoxUsername(t testutils.T) string {
	return stringutils.FirstNonEmpty(mustGetEnvVarInCI(t, usernameEnvVar), defaultUsername)
}

// RoxPassword returns the rox basic auth password (or fails the test if not found).
func RoxPassword(t testutils.T) string {
	if pw := mustGetEnvVarInCI(t, env.PasswordEnv.EnvVar()); pw != "" {
		return pw
	}

	pwFromFileBytes, err := os.ReadFile(filepath.Join(testutils.GetTestWorkspaceDir(t), defaultPasswordPath))
	require.NoErrorf(t, err, "no password set via %s, and could not read password file")

	return strings.TrimSpace(string(pwFromFileBytes))
}

// RoxAPIEndpoint returns the central API endpoint (or fails the test if not found).
func RoxAPIEndpoint(t testutils.T) string {
	return stringutils.FirstNonEmpty(mustGetEnvVarInCI(t, apiEndpointEnvVar), defaultAPIEndpoint)
}

// UnauthenticatedGRPCConnectionToCentral is like GRPCConnectionToCentral,
// but does not inject credentials into the request.
func UnauthenticatedGRPCConnectionToCentral(t *testing.T) *grpc.ClientConn {
	return grpcConnectionToCentral(t, nil)
}

// GRPCConnectionToCentral returns a GRPC connection to Central, which can be used in E2E tests.
// Basic auth is used to establish the connection if no other options are provided which
// configure auth. It fatals the test if there's an error.
func GRPCConnectionToCentral(t testutils.T, optsFuncs ...func(opts *clientconn.Options)) *grpc.ClientConn {
	var tmpOpts clientconn.Options
	for _, optsFunc := range optsFuncs {
		optsFunc(&tmpOpts)
	}

	if tmpOpts.PerRPCCreds == nil {
		// No options configured auth, fallback to basic auth.
		optsFuncs = append(optsFuncs, func(opts *clientconn.Options) {
			opts.ConfigureBasicAuth(RoxUsername(t), RoxPassword(t))
		})
	}

	return grpcConnectionToCentral(t, optsFuncs...)
}

// shouldRetryForTests determines if a gRPC error should be retried in tests.
// This is based on the retry logic from roxctl/common/connection.go.
func shouldRetryForTests(err error) bool {
	if grpcErr, ok := status.FromError(err); ok {
		code := grpcErr.Code()
		// Retry on common transient errors
		if code == codes.DeadlineExceeded || code == codes.Unavailable || code == codes.ResourceExhausted {
			return true
		}
	}
	return false
}

// loggingUnaryInterceptor logs gRPC requests and responses for debugging test failures.
func loggingUnaryInterceptor(logger testutils.T) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)

		if err != nil {
			logger.Logf("gRPC call %s failed after %v: %v", method, duration, err)
		} else {
			logger.Logf("gRPC call %s succeeded in %v", method, duration)
		}

		return err
	}
}

// loggingStreamInterceptor logs gRPC stream requests for debugging test failures.
func loggingStreamInterceptor(logger testutils.T) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()
		stream, err := streamer(ctx, desc, cc, method, opts...)
		duration := time.Since(start)

		if err != nil {
			logger.Logf("gRPC stream %s failed after %v: %v", method, duration, err)
		} else {
			logger.Logf("gRPC stream %s established in %v", method, duration)
		}

		return stream, err
	}
}

func grpcConnectionToCentral(t testutils.T, optsFuncs ...func(options *clientconn.Options)) *grpc.ClientConn {
	endpoint := RoxAPIEndpoint(t)
	host, _, _, err := netutil.ParseEndpoint(endpoint)
	require.NoError(t, err)

	opts := clientconn.Options{
		TLS: clientconn.TLSConfigOptions{
			InsecureSkipVerify: true,
			ServerName:         host,
		},
	}

	for _, optsFunc := range optsFuncs {
		optsFunc(&opts)
	}

	// Add retry interceptors similar to roxctl to handle transient failures
	const initialBackoffDuration = 100 * time.Millisecond
	retryOpts := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(initialBackoffDuration)),
		// First retry after 100ms, max 5 retries for tests (less aggressive than roxctl's 10)
		grpc_retry.WithMax(5),
		grpc_retry.WithRetriable(shouldRetryForTests),
	}

	grpcDialOpts := []grpc.DialOption{}

	// Add logging interceptors if the test object supports logging (e.g., *testing.T)
	// Logging interceptor first, then retry - this ensures we log each retry attempt
	grpcDialOpts = append(grpcDialOpts,
		grpc.WithChainUnaryInterceptor(
			loggingUnaryInterceptor(t),
			grpc_retry.UnaryClientInterceptor(retryOpts...),
		),
		grpc.WithChainStreamInterceptor(
			loggingStreamInterceptor(t),
			grpc_retry.StreamClientInterceptor(retryOpts...),
		),
	)

	conn, err := clientconn.GRPCConnection(context.Background(), mtls.CentralSubject, endpoint, opts, grpcDialOpts...)
	require.NoError(t, err)
	return conn
}

// HTTPClientForCentral returns an *http.Client for talking to central in tests. Basic auth credentials and
// the hostname and scheme part of the URL may be omitted.
func HTTPClientForCentral(t testutils.T) *http.Client {
	baseTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	endpoint := RoxAPIEndpoint(t)
	user := RoxUsername(t)
	pw := RoxPassword(t)

	transport := httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		modReq := req.Clone(req.Context())
		modReq.SetBasicAuth(user, pw)
		if modReq.URL.Host == "" {
			modReq.URL.Host = endpoint
		}
		if modReq.URL.Scheme == "" {
			modReq.URL.Scheme = "https"
		}
		resp, err := baseTransport.RoundTrip(modReq)
		if err != nil {
			return nil, err
		}
		resp.Request = req
		return resp, nil
	})

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	return client
}
