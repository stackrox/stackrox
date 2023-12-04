package service

import (
	"context"

	imageIntegrationStore "github.com/stackrox/rox/central/imageintegration/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	v1.CredentialExpiryServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(imageIntegrations imageIntegrationStore.DataStore) Service {
	scannerTLSConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		// This case is hit if the Central CA cert cannot be loaded. This case is hit during some upgrade-tests
		// because in ancient versions, Central used to issue itself a cert on startup.
		// However, Central uses the exact same function to talk to scanner, so any customer who actually uses
		// scanner must have patched their deployment to not hit this.
		// At the same time, we don't want to make this a fatal error, so just log a warning.
		log.Warnf("Failed to initialize scanner TLS config: %v", err)
		scannerTLSConfig = nil
	}
	scannerV4IndexerTLSConfig, err := clientconn.TLSConfig(mtls.ScannerV4IndexerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		log.Warnf("Failed to initialize Scanner V4 Indexer TLS config: %v", err)
		scannerV4IndexerTLSConfig = nil
	}
	scannerV4MatcherTLSConfig, err := clientconn.TLSConfig(mtls.ScannerV4MatcherSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		log.Warnf("Failed to initialize Scanner V4 Matcher TLS config: %v", err)
		scannerV4MatcherTLSConfig = nil
	}
	scannerV4DBTLSConfig, err := clientconn.TLSConfig(mtls.ScannerV4DBSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		log.Warnf("Failed to initialize Scanner V4 DB TLS config: %v", err)
		scannerV4DBTLSConfig = nil
	}

	return &serviceImpl{
		imageIntegrations:      imageIntegrations,
		scannerConfig:          scannerTLSConfig,
		scannerV4IndexerConfig: scannerV4IndexerTLSConfig,
		scannerV4MatcherConfig: scannerV4MatcherTLSConfig,
		scannerV4DBConfig:      scannerV4DBTLSConfig,
	}
}
