package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/mitre/datastore"
)

// Service provides the MITRE ATTACK service interface.
type Service interface {
	grpc.APIService

	v1.MitreAttackServiceServer
}

// New returns a new MITRE ATTACK Service instance.
func New(store datastore.AttackReadOnlyDataStore) Service {
	return &serviceImpl{
		store: store,
	}
}
