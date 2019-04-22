package roles

import (
	"fmt"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srole/datastore"
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

// NewPipeline returns a new instance of Pipeline for k8s role
func NewPipeline(clusters clusterDatastore.DataStore, roles datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusters:        clusters,
		roles:           roles,
		reconcileStore:  reconciliation.NewStore(),
		riskReprocessor: reprocessor.Singleton(),
	}
}

type pipelineImpl struct {
	clusters        clusterDatastore.DataStore
	roles           datastore.DataStore
	reconcileStore  reconciliation.Store
	riskReprocessor reprocessor.Loop
}

func (s *pipelineImpl) Reconcile(clusterID string) error {

	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.roles.Search(query)
	if err != nil {
		return err
	}

	err = reconciliation.Perform(s.reconcileStore, search.ResultsToIDSet(results), "k8sroles", func(id string) error {
		return s.runRemovePipeline(central.ResourceAction_REMOVE_RESOURCE, &storage.K8SRole{Id: id})
	})

	if err != nil {
		return err
	}

	s.riskReprocessor.ReprocessRisk()
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetRole() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	if !features.K8sRBAC.Enabled() {
		return nil
	}

	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Role)

	event := msg.GetEvent()
	role := event.GetRole()
	role.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event.GetAction(), role)
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE:
		s.reconcileStore.Add(event.GetId())
		return s.runGeneralPipeline(event.GetAction(), role)
	default:
		return fmt.Errorf("event action '%s' for k8s role does not exist", event.GetAction())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action central.ResourceAction, event *storage.K8SRole) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the k8s role from persistence depending on the event action.
	if err := s.roles.RemoveRole(event.GetId()); err != nil {
		return err
	}

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action central.ResourceAction, role *storage.K8SRole) error {
	if err := s.validateInput(role); err != nil {
		return err
	}

	if err := s.enrichCluster(role); err != nil {
		return err
	}

	if err := s.roles.UpsertRole(role); err != nil {
		return err
	}

	return nil
}

func (s *pipelineImpl) validateInput(role *storage.K8SRole) error {
	// validate input.
	if role == nil {
		return fmt.Errorf("role must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(role *storage.K8SRole) error {
	role.ClusterName = ""

	cluster, clusterExists, err := s.clusters.GetCluster(role.GetClusterId())
	switch {
	case err != nil:
		log.Errorf("Couldn't get name of cluster: %v", err)
		return err
	case !clusterExists:
		log.Errorf("Couldn't find cluster '%q'", role.GetClusterId())
		return err
	default:
		role.ClusterName = cluster.GetName()
	}
	return nil
}

func (s *pipelineImpl) OnFinish(clusterID string) {}
