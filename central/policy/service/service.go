package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/detection/lifecycle"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/searchbasedpolicies/matcher"
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
	notifiers notifierDataStore.DataStore,
	reprocessor reprocessor.Loop,
	buildTimePolicies detection.PolicySet,
	deploymentMatcherBuilder matcher.Builder,
	imageMatcherBuilder matcher.Builder,
	manager lifecycle.Manager,
	processor notifierProcessor.Processor,
	metadataCache expiringcache.Cache,
	scanCache expiringcache.Cache) Service {
	return &serviceImpl{
		policies:    policies,
		clusters:    clusters,
		deployments: deployments,
		reprocessor: reprocessor,

		buildTimePolicies: buildTimePolicies,
		testMatchBuilder:  deploymentMatcherBuilder,
		lifecycleManager:  manager,

		processor: processor,

		metadataCache: metadataCache,
		scanCache:     scanCache,

		validator: newPolicyValidator(notifiers, clusters, deploymentMatcherBuilder, imageMatcherBuilder),
	}
}
