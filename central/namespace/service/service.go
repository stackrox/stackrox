package service

import (
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/namespace/datastore"
	npDS "github.com/stackrox/stackrox/central/networkpolicies/datastore"
	secretDataStore "github.com/stackrox/stackrox/central/secret/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service is the GRPC service interface that provides the entry point for processing deployment events.
type Service interface {
	grpc.APIService
	v1.NamespaceServiceServer
}

// New returns a new instance of service.
func New(datastore datastore.DataStore, deployments deploymentDataStore.DataStore, secrets secretDataStore.DataStore, networkPolicies npDS.DataStore) Service {
	return &serviceImpl{
		datastore:       datastore,
		deployments:     deployments,
		secrets:         secrets,
		networkPolicies: networkPolicies,
	}
}
