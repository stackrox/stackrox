package service

import (
	"context"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/detection/lifecycle"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/backgroundtasks"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc"
	mitreDS "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/notifier"
)

// Service provides the interface to the microservice that serves policy data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.PolicyServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(policies datastore.DataStore,
	clusters clusterDataStore.DataStore,
	deployments deploymentDataStore.DataStore,
	networkPolicies networkPolicyDS.DataStore,
	notifiers notifierDataStore.DataStore,
	mitreStore mitreDS.AttackReadOnlyDataStore,
	reprocessor reprocessor.Loop,
	buildTimePolicies detection.PolicySet,
	manager lifecycle.Manager,
	processor notifier.Processor,
	metadataCache expiringcache.Cache,
	connectionManager connection.Manager) Service {
	backgroundTaskManager := backgroundtasks.NewManager()
	backgroundTaskManager.Start()
	return &serviceImpl{
		policies:          policies,
		clusters:          clusters,
		deployments:       deployments,
		reprocessor:       reprocessor,
		notifiers:         notifiers,
		mitreStore:        mitreStore,
		buildTimePolicies: buildTimePolicies,
		lifecycleManager:  manager,
		connectionManager: connectionManager,
		networkPolicies:   networkPolicies,

		processor: processor,

		metadataCache: metadataCache,

		validator:              newPolicyValidator(notifiers),
		dryRunPolicyJobManager: backgroundTaskManager,
	}
}
