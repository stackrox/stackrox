package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/centralproxy/pkg/server"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/client/authn/tokenbased"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/securedcluster"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	log = logging.LoggerForModule()
)

const (
	defaultPort                = "8080"
	defaultCentralEndpoint     = "central.stackrox.svc:443"
	defaultCertificatePath     = "/etc/ssl/certs"
	defaultNamespace           = "stackrox"
	shutdownTimeout            = 30 * time.Second
)

func main() {
	var (
		port             = flag.String("port", env.RegisterSetting("CENTRAL_PROXY_PORT", defaultPort).Setting(), "Port to listen on")
		centralEndpoint  = flag.String("central-endpoint", env.RegisterSetting("CENTRAL_ENDPOINT", defaultCentralEndpoint).Setting(), "Central API endpoint")
		certificatePath  = flag.String("certificate-path", env.RegisterSetting("CERTIFICATE_PATH", defaultCertificatePath).Setting(), "Path to TLS certificates")
		namespace        = flag.String("namespace", env.RegisterSetting("NAMESPACE", defaultNamespace).Setting(), "Kubernetes namespace")
	)
	flag.Parse()

	if err := run(*port, *centralEndpoint, *certificatePath, *namespace); err != nil {
		log.Fatalf("Central Proxy failed: %v", err)
	}
}

func run(port, centralEndpoint, certificatePath, namespace string) error {
	log.Infof("Starting Central Proxy on port %s", port)
	log.Infof("Central endpoint: %s", centralEndpoint)
	log.Infof("Certificate path: %s", certificatePath)
	log.Infof("Namespace: %s", namespace)

	// Create Kubernetes client for RBAC checks
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to create Kubernetes config")
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create Kubernetes client")
	}

	// Load Central Proxy TLS certificates
	tlsConfig, err := loadTLSConfig(certificatePath)
	if err != nil {
		return errors.Wrap(err, "failed to load TLS configuration")
	}

	// Create Central API client
	centralClient, err := createCentralClient(centralEndpoint, tlsConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create Central client")
	}

	// Create and configure the server
	srv := server.New(&server.Config{
		Port:          port,
		K8sClient:     k8sClient,
		CentralClient: centralClient,
		Namespace:     namespace,
	})

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:      ":" + port,
		Handler:   srv,
		TLSConfig: tlsConfig,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Infof("Central Proxy listening on :%s", port)
		serverErrors <- httpServer.ListenAndServeTLS("", "")
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server failed")

	case sig := <-interrupt:
		log.Infof("Received signal %v, shutting down", sig)

		// Attempt graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Errorf("Failed to gracefully shutdown server: %v", err)
			return httpServer.Close()
		}

		log.Info("Server gracefully stopped")
	}

	return nil
}

func loadTLSConfig(certificatePath string) (*tls.Config, error) {
	// Load Central Proxy certificates from the secret mounted at certificatePath
	certFile := fmt.Sprintf("%s/%s", certificatePath, mtls.ServiceCertFileName)
	keyFile := fmt.Sprintf("%s/%s", certificatePath, mtls.ServiceKeyFileName)
	caFile := fmt.Sprintf("%s/%s", certificatePath, mtls.CACertFileName)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load certificate pair")
	}

	// Load CA certificate for client verification
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	// Add CA to certificate pool
	if err := mtls.LoadCustomCAs(tlsConfig, caCert); err != nil {
		return nil, errors.Wrap(err, "failed to load CA certificates")
	}

	return tlsConfig, nil
}

func createCentralClient(centralEndpoint string, tlsConfig *tls.Config) (interface{}, error) {
	// TODO: Implement Central API client with mTLS
	// This will be a gRPC client that connects to Central's API
	log.Infof("Creating Central client for endpoint: %s", centralEndpoint)
	
	// Placeholder - will be implemented with actual Central gRPC client
	return nil, nil
}