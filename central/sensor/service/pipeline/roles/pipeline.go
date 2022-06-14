package roles

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srole/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
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
		riskReprocessor: reprocessor.Singleton(),
	}
}

type pipelineImpl struct {
	clusters        clusterDatastore.DataStore
	roles           datastore.DataStore
	riskReprocessor reprocessor.Loop
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.roles.Search(ctx, query)
	if err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_Role)(nil))
	err = reconciliation.Perform(store, search.ResultsToIDSet(results), "k8sroles", func(id string) error {
		return s.runRemovePipeline(ctx, central.ResourceAction_REMOVE_RESOURCE, &storage.K8SRole{Id: id})
	})

	if err != nil {
		return err
	}
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetRole() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Role)

	event := msg.GetEvent()
	role := event.GetRole()
	role.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(ctx, event.GetAction(), role)
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE:
		return s.runGeneralPipeline(ctx, event.GetAction(), role)
	default:
		return fmt.Errorf("event action '%s' for k8s role does not exist", event.GetAction())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, action central.ResourceAction, event *storage.K8SRole) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the k8s role from persistence depending on the event action.
	if err := s.roles.RemoveRole(ctx, event.GetId()); err != nil {
		return err
	}

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, action central.ResourceAction, role *storage.K8SRole) error {
	if err := s.validateInput(role); err != nil {
		return err
	}

	if err := s.enrichCluster(ctx, role); err != nil {
		return err
	}

	if err := s.roles.UpsertRole(ctx, role); err != nil {
		return err
	}

	return nil
}

func (s *pipelineImpl) validateInput(role *storage.K8SRole) error {
	// validate input.
	if role == nil {
		return errors.New("role must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(ctx context.Context, role *storage.K8SRole) error {
	role.ClusterName = ""

	clusterName, clusterExists, err := s.clusters.GetClusterName(ctx, role.GetClusterId())
	switch {
	case err != nil:
		log.Errorf("Couldn't get name of cluster: %v", err)
		return err
	case !clusterExists:
		log.Errorf("Couldn't find cluster '%q'", role.GetClusterId())
		return err
	default:
		role.ClusterName = clusterName
	}
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
