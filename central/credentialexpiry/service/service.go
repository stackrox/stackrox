package service

import (
	"context"

	imageIntegrationStore "github.com/stackrox/stackrox/central/imageintegration/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/clientconn"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/mtls"
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
	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		// This case is hit if the Central CA cert cannot be loaded. This case is hit during some upgrade-tests
		// because in ancient versions, Central used to issue itself a cert on startup.
		// However, Central uses the exact same function to talk to scanner, so any customer who actually uses
		// scanner must have patched their deployment to not hit this.
		// At the same time, we don't want to make this a fatal error, so just log a warning.
		log.Warnf("Failed to initialize scanner TLS config: %v", err)
		tlsConfig = nil
	}

	return &serviceImpl{imageIntegrations: imageIntegrations, scannerConfig: tlsConfig}
}
