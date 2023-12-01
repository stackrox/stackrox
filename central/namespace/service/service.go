package service

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace/datastore"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
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
