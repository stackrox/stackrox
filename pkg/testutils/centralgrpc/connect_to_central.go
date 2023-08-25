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

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	usernameEnvVar = "ROX_USERNAME"
	passwordEnvVar = "ROX_PASSWORD"

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
	if pw := mustGetEnvVarInCI(t, passwordEnvVar); pw != "" {
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
// It fatals the test if there's an error.
func GRPCConnectionToCentral(t testutils.T) *grpc.ClientConn {
	return grpcConnectionToCentral(t, func(opts *clientconn.Options) {
		opts.ConfigureBasicAuth(RoxUsername(t), RoxPassword(t))
	})
}

func grpcConnectionToCentral(t testutils.T, optsModifyFunc func(options *clientconn.Options)) *grpc.ClientConn {
	endpoint := RoxAPIEndpoint(t)
	host, _, _, err := netutil.ParseEndpoint(endpoint)
	require.NoError(t, err)

	opts := clientconn.Options{
		TLS: clientconn.TLSConfigOptions{
			InsecureSkipVerify: true,
			ServerName:         host,
		},
	}
	if optsModifyFunc != nil {
		optsModifyFunc(&opts)
	}
	conn, err := clientconn.GRPCConnection(context.Background(), mtls.CentralSubject, endpoint, opts)
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
