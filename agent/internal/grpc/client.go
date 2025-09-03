package grpc

import (
	"context"
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"github.com/stackrox/rox/generated/internalapi/sensor"
)

// Config holds gRPC client configuration
type Config struct {
	SensorURL  string
	CertPath   string // Legacy cert path
	CACertFile string // From ROX_MTLS_CA_FILE env var
	ClientCert string // From ROX_MTLS_CERT_FILE env var
	ClientKey  string // From ROX_MTLS_KEY_FILE env var
}

// UserAgentInterceptor adds the "Rox VM Agent" user-agent header
type UserAgentInterceptor struct{}

// Unary implements grpc.UnaryClientInterceptor
func (i *UserAgentInterceptor) Unary(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	ctx = metadata.AppendToOutgoingContext(ctx, "user-agent", "Rox VM Agent")
	return invoker(ctx, method, req, reply, cc, opts...)
}

// createClient creates a gRPC client connection with TLS
func createClient(config Config) (sensor.VirtualMachineIndexReportServiceClient, *grpc.ClientConn, error) {
	var cert tls.Certificate
	var err error

	// Determine which certificate loading method to use
	if config.CertPath != "" {
		// Legacy mode: load from cert-path directory
		cert, err = tls.LoadX509KeyPair(
			fmt.Sprintf("%s/cert.pem", config.CertPath),
			fmt.Sprintf("%s/key.pem", config.CertPath),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load client certificates from cert-path: %w", err)
		}
	} else {
		// ROX_MTLS mode: load from environment variable paths
		cert, err = tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load client certificates from ROX_MTLS paths: %w", err)
		}
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "sensor.stackrox.svc",
	}

	// Load CA certificate
	var caCertPath string
	if config.CertPath != "" {
		// Legacy mode
		caCertPath = fmt.Sprintf("%s/ca.pem", config.CertPath)
	} else {
		// ROX_MTLS mode
		caCertPath = config.CACertFile
	}

	if caCert, err := loadCACertificate(caCertPath); err == nil {
		tlsConfig.RootCAs = caCert
	} else {
		// Log but continue - will use system CA pool
		log.Warnf("Failed to load CA certificate from %s, using system CA pool: %v", caCertPath, err)
	}

	// Create gRPC connection with TLS
	conn, err := grpc.Dial(config.SensorURL,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithUnaryInterceptor((&UserAgentInterceptor{}).Unary),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to sensor: %w", err)
	}

	client := sensor.NewVirtualMachineIndexReportServiceClient(conn)
	return client, conn, nil
}
