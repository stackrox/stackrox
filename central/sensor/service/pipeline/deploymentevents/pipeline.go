package deploymentevents

import (
	"context"
	"time"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentutils "github.com/stackrox/rox/central/deployment/utils"
	"github.com/stackrox/rox/central/detection/lifecycle"
	countMetrics "github.com/stackrox/rox/central/metrics"
	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		lifecycle.SingletonManager(),
		graph.Singleton(),
		reprocessor.Singleton(),
		networkBaselineManager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(
	clusters clusterDataStore.DataStore,
	deployments deploymentDataStore.DataStore,
	manager lifecycle.Manager,
	graphEvaluator graph.Evaluator,
	reprocessor reprocessor.Loop,
	networkBaselines networkBaselineManager.Manager,
) pipeline.Fragment {
	return &pipelineImpl{
		validateInput:     newValidateInput(),
		clusterEnrichment: newClusterEnrichment(clusters),
		lifecycleManager:  manager,

		graphEvaluator:   graphEvaluator,
		deployments:      deployments,
		clusters:         clusters,
		networkBaselines: networkBaselines,

		reprocessor: reprocessor,
	}
}

type pipelineImpl struct {
	// pipeline stages.
	validateInput     *validateInputImpl
	clusterEnrichment *clusterEnrichmentImpl
	lifecycleManager  lifecycle.Manager

	deployments      deploymentDataStore.DataStore
	clusters         clusterDataStore.DataStore
	networkBaselines networkBaselineManager.Manager
	reprocessor      reprocessor.Loop

	graphEvaluator graph.Evaluator
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.deployments.Search(ctx, query)
	if err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_Deployment)(nil))
	return reconciliation.Perform(store, search.ResultsToIDSet(results), "deployments", func(id string) error {
		return s.runRemovePipeline(ctx, id, clusterID, true)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetDeployment() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Deployment)

	event := msg.GetEvent()
	deployment := event.GetDeployment()
	deployment.ClusterId = clusterID

	// ROX-22002: Remove invalid null characters in annotations
	stringutils.SanitizeMapValues(deployment.GetAnnotations())

	var err error
	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		err = s.runRemovePipeline(ctx, deployment.GetId(), clusterID, false)
	default:
		err = s.runGeneralPipeline(ctx, deployment, event.GetAction())
	}
	return err
}

// runRemovePipeline handles the removal of a deployment, routing through the tombstone
// soft-delete path when the DeploymentTombstones feature is enabled and a TTL is configured,
// or falling back to hard-delete (original behavior) otherwise.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, deploymentID, clusterID string, isReconciliation bool) error {
	if features.DeploymentTombstones.Enabled() {
		ttlDays := s.getTombstoneTTL(ctx)
		if ttlDays > 0 {
			// Tombstone path: transition alert state and soft-delete the deployment.
			// For reconciliation the alerts pipeline won't fire, so always call here.
			// For live removes the alerts pipeline is guarded to skip DeploymentRemoved
			// when tombstoning is active, so calling DeploymentTombstoned here is safe.
			if err := s.lifecycleManager.DeploymentTombstoned(deploymentID); err != nil {
				return err
			}

			// Clean up network baselines before the soft-delete so that a failed
			// baseline cleanup cannot leave stale edges after the deployment is gone.
			if err := s.networkBaselines.ProcessDeploymentDelete(deploymentID); err != nil {
				return err
			}

			expiresAt := time.Now().Add(time.Duration(ttlDays) * 24 * time.Hour)
			if err := s.deployments.TombstoneDeployment(ctx, clusterID, deploymentID, expiresAt); err != nil {
				return err
			}

			s.graphEvaluator.IncrementEpoch(clusterID)
			return nil
		}
	}

	// Hard-delete path (original behavior).
	// If we're in reconciliation, manage the alert lifecycle here because sensor does not
	// send a separate remove event for reconciled deployments. For live removes the alerts
	// pipeline handles this sequentially to avoid a race.
	if isReconciliation {
		if err := s.lifecycleManager.DeploymentRemoved(deploymentID); err != nil {
			return err
		}
	}

	// Before removing the deployment, clean up all the network baselines that had an edge
	// to this deployment. Doing this first ensures baselines are not left stale if the
	// deployment removal succeeds but the baseline cleanup does not run.
	if err := s.networkBaselines.ProcessDeploymentDelete(deploymentID); err != nil {
		return err
	}

	// Remove the deployment from persistence.
	if err := s.deployments.RemoveDeployment(ctx, clusterID, deploymentID); err != nil {
		return err
	}

	s.graphEvaluator.IncrementEpoch(clusterID)
	return nil
}

// getTombstoneTTL reads the tombstone retention duration from the system configuration.
// It returns the configured TTL in days, or DefaultTombstoneRetentionDays if the config
// cannot be read or has no explicit value set.
func (s *pipelineImpl) getTombstoneTTL(ctx context.Context) int32 {
	// Use elevated access because reading global system config is an admin-level operation
	// that must not be filtered by the sensor-facing request context's SAC scope.
	cfgCtx := sac.WithAllAccess(ctx)
	pvtConfig, err := configDatastore.Singleton().GetPrivateConfig(cfgCtx)
	if err != nil || pvtConfig == nil {
		log.Warnf("Failed to read private config for tombstone TTL, using default %d days: %v",
			configDatastore.DefaultTombstoneRetentionDays, err)
		return configDatastore.DefaultTombstoneRetentionDays
	}
	ttl := pvtConfig.GetTombstoneRetentionConfig().GetRetentionDurationDays()
	if ttl == 0 {
		return configDatastore.DefaultTombstoneRetentionDays
	}
	return ttl
}

func compareMap(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		if v2, ok := m2[k]; !ok || v2 != v1 {
			return false
		}
	}
	return true
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, deployment *storage.Deployment, action central.ResourceAction) error {
	// Validate the deployment we receive has necessary fields set.
	if err := s.validateInput.do(deployment); err != nil {
		return err
	}

	// Fill in cluster information.
	if err := s.clusterEnrichment.do(ctx, deployment); err != nil {
		log.Errorf("Couldn't get cluster identity: %s", err)
		return err
	}

	if features.FlattenImageData.Enabled() {
		// IDV2s may not be set if sensor is running an older version
		deploymentutils.PopulateContainerImageIDV2s(deployment)
	}

	incrementNetworkGraphEpoch := true
	// Only need to get if it's an update call
	if action == central.ResourceAction_UPDATE_RESOURCE || action == central.ResourceAction_SYNC_RESOURCE {
		oldDeployment, exists, err := s.deployments.GetDeployment(ctx, deployment.GetId())
		if err != nil {
			return err
		}
		// If it exists, check to see if we can dedupe it
		if exists {
			if oldDeployment.GetHash() == deployment.GetHash() {
				// There is a separate handler for ContainerInstances,
				// so there is no longer a need to continue from this point.
				// This will only be reached upon a re-sync event from k8s.
				return nil
			}
			incrementNetworkGraphEpoch = !compareMap(oldDeployment.GetPodLabels(), deployment.GetPodLabels())
		}
	}

	// Add/Update the deployment from persistence depending on the deployment action.
	if err := s.deployments.UpsertDeployment(ctx, deployment); err != nil {
		return err
	}

	// Inform network baseline manager that a new deployment has been created
	if err := s.networkBaselines.ProcessDeploymentCreate(
		deployment.GetId(),
		deployment.GetName(),
		deployment.GetClusterId(),
		deployment.GetNamespace(),
	); err != nil {
		return err
	}

	// Update risk asynchronously
	s.reprocessor.ReprocessRiskForDeployments(deployment.GetId())

	if incrementNetworkGraphEpoch {
		s.graphEvaluator.IncrementEpoch(deployment.GetClusterId())
	}
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
