package service

import (
	"github.com/stackrox/stackrox/central/mitre/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service provides the MITRE ATTACK service interface.
type Service interface {
	grpc.APIService

	v1.MitreAttackServiceServer
}

// New returns a new MITRE ATTACK Service instance.
func New(store datastore.MitreAttackReadOnlyDataStore) Service {
	return &serviceImpl{
		store: store,
	}
}
