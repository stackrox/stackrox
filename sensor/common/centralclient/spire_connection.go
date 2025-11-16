package centralclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"github.com/stackrox/rox/pkg/mtls"
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

	// Get the SVID and trust bundle
	svid, err := source.GetX509SVID()
	if err != nil {
		_ = source.Close()
		return nil, errors.Wrap(err, "failed to get X509-SVID from source")
	}

	// Create a custom TLS config that:
	// 1. Uses SPIRE SVID as client certificate (for Sensor's identity)
	// 2. Trusts both SPIRE trust bundle AND traditional mTLS CA (for backward compat)

	// Get the trust bundle for verification
	bundle, err := source.GetX509BundleForTrustDomain(svid.ID.TrustDomain())
	if err != nil {
		_ = source.Close()
		return nil, errors.Wrap(err, "failed to get X509 bundle from SPIRE")
	}

	// Create root CA pool that includes both SPIRE and traditional CAs
	rootCAs := x509.NewCertPool()

	// Add SPIRE trust bundle
	for _, cert := range bundle.X509Authorities() {
		rootCAs.AddCert(cert)
	}

	// Also add traditional mTLS CA for backward compatibility
	serviceCA, err := mtls.CACertPEM()
	if err != nil {
		log.Warnf("SPIRE: Failed to get traditional service CA: %v", err)
	} else if len(serviceCA) > 0 {
		if !rootCAs.AppendCertsFromPEM(serviceCA) {
			log.Warn("SPIRE: Failed to add traditional service CA to root pool")
		} else {
			log.Info("SPIRE: Added traditional service CA for backward compatibility")
		}
	}

	// Marshal certificates to DER format for tls.Certificate
	var certDER [][]byte
	for _, cert := range svid.Certificates {
		certDER = append(certDER, cert.Raw)
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    rootCAs,
		// Use SPIRE SVID as client certificate
		Certificates: []tls.Certificate{
			{
				Certificate: certDER,
				PrivateKey:  svid.PrivateKey,
			},
		},
	}

	// Create gRPC connection with hybrid TLS credentials
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
