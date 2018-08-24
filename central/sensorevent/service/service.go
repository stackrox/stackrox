package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/networkgraph"
	networkPolicyStore "github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/central/risk"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/deploymentevents"
	namespacePipeline "github.com/stackrox/rox/central/sensorevent/service/pipeline/namespaces"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/networkpolicies"
	deploymentEventStore "github.com/stackrox/rox/central/sensorevent/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/net/context"
)

var (
	log = logging.LoggerForModule()
)

// Service is the GRPC service interface that provides the entry point for processing deployment events.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	RecordEvent(stream v1.SensorEventService_RecordEventServer) error
}

// New returns a new instance of service.
func New(detector deployTimeDetection.Detector,
	scorer risk.Scorer,
	deploymentEvents deploymentEventStore.Store,
	images imageDataStore.DataStore,
	deployments deploymentDataStore.DataStore,
	clusters clusterDataStore.DataStore,
	networkPolicies networkPolicyStore.Store,
	namespaces namespaceDataStore.Store,
	graphEvaluator networkgraph.Evaluator) Service {
	return &serviceImpl{
		detector: detector,
		scorer:   scorer,

		deploymentEvents: deploymentEvents,
		images:           images,
		deployments:      deployments,
		clusters:         clusters,
		networkPolicies:  networkPolicies,
		namespaces:       namespaces,

		deploymentPipeline:    deploymentevents.NewPipeline(clusters, deployments, images, detector, graphEvaluator),
		networkPolicyPipeline: networkpolicies.NewPipeline(clusters, networkPolicies, graphEvaluator),
		namespacePipeline:     namespacePipeline.NewPipeline(clusters, namespaces, graphEvaluator),
	}
}
