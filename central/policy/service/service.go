package service

import (
	"context"

	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/detection"
	"github.com/stackrox/stackrox/central/detection/lifecycle"
	mitreDataStore "github.com/stackrox/stackrox/central/mitre/datastore"
	networkPolicyDS "github.com/stackrox/stackrox/central/networkpolicies/datastore"
	notifierDataStore "github.com/stackrox/stackrox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/stackrox/central/notifier/processor"
	"github.com/stackrox/stackrox/central/policy/datastore"
	"github.com/stackrox/stackrox/central/reprocessor"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/backgroundtasks"
	"github.com/stackrox/stackrox/pkg/expiringcache"
	"github.com/stackrox/stackrox/pkg/grpc"
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
	mitreStore mitreDataStore.MitreAttackReadOnlyDataStore,
	reprocessor reprocessor.Loop,
	buildTimePolicies detection.PolicySet,
	manager lifecycle.Manager,
	processor notifierProcessor.Processor,
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
