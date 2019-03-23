package service

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globalstore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
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
func New(alerts alertDataStore.DataStore, deployments deploymentDataStore.DataStore, images imageDataStore.DataStore, policies policyDataStore.DataStore,
	secrets secretDataStore.DataStore, serviceAccounts serviceAccountDataStore.DataStore, nodes nodeDataStore.GlobalStore, namespaces namespaceDataStore.DataStore, aggregator aggregation.Aggregator) Service {
	s := &serviceImpl{
		alerts:          alerts,
		deployments:     deployments,
		images:          images,
		policies:        policies,
		secrets:         secrets,
		serviceAccounts: serviceAccounts,
		nodes:           nodes,
		namespaces:      namespaces,
		aggregator:      aggregator,
	}
	s.initializeAuthorizer()
	return s
}
