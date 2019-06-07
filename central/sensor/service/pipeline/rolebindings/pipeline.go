package rolebindings

import (
	"context"
	"fmt"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDatastore.Singleton(), datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline for k8s role bindings
func NewPipeline(clusters clusterDatastore.DataStore, bindings datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusters:        clusters,
		bindings:        bindings,
		reconcileStore:  reconciliation.NewStore(),
		riskReprocessor: reprocessor.Singleton(),
	}
}

type pipelineImpl struct {
	clusters        clusterDatastore.DataStore
	bindings        datastore.DataStore
	reconcileStore  reconciliation.Store
	riskReprocessor reprocessor.Loop
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.bindings.Search(ctx, query)
	if err != nil {
		return err
	}

	err = reconciliation.Perform(s.reconcileStore, search.ResultsToIDSet(results), "k8srolebindings", func(id string) error {
		return s.runRemovePipeline(ctx, central.ResourceAction_REMOVE_RESOURCE, &storage.K8SRoleBinding{Id: id})
	})

	if err != nil {
		return err
	}

	s.riskReprocessor.ReprocessRisk()
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetBinding() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	if !features.K8sRBAC.Enabled() {
		return nil
	}

	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.RoleBinding)

	event := msg.GetEvent()
	binding := event.GetBinding()
	binding.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(ctx, event.GetAction(), binding)
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE:
		s.reconcileStore.Add(event.GetId())
		return s.runGeneralPipeline(ctx, event.GetAction(), binding)
	default:
		return fmt.Errorf("Event action '%s' for k8s role binding does not exist", event.GetAction())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, action central.ResourceAction, event *storage.K8SRoleBinding) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the k8s role binding from persistence depending on the event action.
	if err := s.bindings.RemoveRoleBinding(ctx, event.GetId()); err != nil {
		return err
	}

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, action central.ResourceAction, binding *storage.K8SRoleBinding) error {
	if err := s.validateInput(binding); err != nil {
		return err
	}

	if err := s.enrichCluster(ctx, binding); err != nil {
		return err
	}

	if err := s.bindings.UpsertRoleBinding(ctx, binding); err != nil {
		return err
	}

	return nil
}

func (s *pipelineImpl) validateInput(binding *storage.K8SRoleBinding) error {
	// validate input.
	if binding == nil {
		return fmt.Errorf("role binding must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(ctx context.Context, binding *storage.K8SRoleBinding) error {
	binding.ClusterName = ""

	cluster, clusterExists, err := s.clusters.GetCluster(ctx, binding.GetClusterId())
	switch {
	case err != nil:
		log.Errorf("Couldn't get name of cluster: %v", err)
		return err
	case !clusterExists:
		log.Errorf("Couldn't find cluster '%q'", binding.GetClusterId())
		return err
	default:
		binding.ClusterName = cluster.GetName()
	}
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
