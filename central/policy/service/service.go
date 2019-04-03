package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDetection "github.com/stackrox/rox/central/detection/image"
	"github.com/stackrox/rox/central/detection/lifecycle"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/central/policy/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc"
	"golang.org/x/net/context"
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.PolicyServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(policies datastore.DataStore,
	clusters clusterDataStore.DataStore,
	deployments deploymentDataStore.DataStore,
	notifiers notifierStore.Store,
	processes processIndicatorDataStore.DataStore,
	reprocessor reprocessor.Loop,
	buildTimePolicies imageDetection.PolicySet,
	manager lifecycle.Manager,
	processor notifierProcessor.Processor,
	metadataCache expiringcache.Cache,
	scanCache expiringcache.Cache) Service {
	return &serviceImpl{
		policies:    policies,
		clusters:    clusters,
		deployments: deployments,
		processes:   processes,
		reprocessor: reprocessor,

		buildTimePolicies: buildTimePolicies,
		lifecycleManager:  manager,

		processor: processor,

		metadataCache: metadataCache,
		scanCache:     scanCache,

		validator: newPolicyValidator(notifiers, clusters),
	}
}
