package centralclient

import (
	"context"
	"crypto/tls"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	spireSocketPath = "/spire-workload-api/spire-agent.sock"
	spireTimeout    = 10 * time.Second
)

// tryConnectViaSPIRE attempts to create a gRPC connection using SPIRE workload API.
// Returns nil, nil if SPIRE is not available (socket doesn't exist).
// Returns nil, err if SPIRE is available but connection fails.
// Returns conn, nil on success.
func tryConnectViaSPIRE(ctx context.Context, centralEndpoint string) (*grpc.ClientConn, error) {
	// Check if SPIRE socket exists
	if _, err := os.Stat(spireSocketPath); os.IsNotExist(err) {
		log.Debug("SPIRE socket not found, will not attempt SPIRE authentication")
		return nil, nil
	}

	log.Info("🔐 SPIRE: Socket found, attempting SPIRE-based connection to Central")

	// Create SPIRE Workload API client
	spireCtx, cancel := context.WithTimeout(ctx, spireTimeout)
	defer cancel()

	source, err := workloadapi.NewX509Source(spireCtx, workloadapi.WithClientOptions(
		workloadapi.WithAddr("unix://"+spireSocketPath),
	))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create SPIRE X509Source")
	}

	log.Info("✅ SPIRE: Successfully obtained X.509-SVID from SPIRE Workload API")

	// Create TLS config using SPIRE credentials
	tlsConfig := tlsconfig.TLSClientConfig(source, tlsconfig.AuthorizeAny())
	tlsConfig.MinVersion = tls.VersionTLS12

	// Create gRPC connection with SPIRE credentials
	conn, err := grpc.NewClient(
		centralEndpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	)
	if err != nil {
		_ = source.Close()
		return nil, errors.Wrap(err, "failed to create gRPC client with SPIRE credentials")
	}

	log.Infof("🎉 SPIRE: Successfully created gRPC connection to Central at %s", centralEndpoint)

	return conn, nil
}
