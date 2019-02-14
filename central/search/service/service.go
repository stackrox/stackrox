package service

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search/enumregistry"
	"golang.org/x/net/context"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.SearchServiceServer
}

// New returns a search service
func New(alerts alertDataStore.DataStore, deployments deploymentDataStore.DataStore, images imageDataStore.DataStore, policies policyDataStore.DataStore, secrets secretDataStore.DataStore, enumRegistry enumregistry.Registry) Service {
	s := &serviceImpl{
		alerts:       alerts,
		deployments:  deployments,
		images:       images,
		policies:     policies,
		secrets:      secrets,
		enumRegistry: enumRegistry,
	}
	s.initializeAuthorizer()
	return s
}
