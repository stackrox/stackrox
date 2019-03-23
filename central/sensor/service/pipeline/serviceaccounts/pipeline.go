package serviceaccounts

import (
	"fmt"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline for service accounts
func NewPipeline(clusters clusterDataStore.DataStore, serviceaccounts datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusters:        clusters,
		serviceaccounts: serviceaccounts,
		reconcileStore:  reconciliation.NewStore(),
	}
}

type pipelineImpl struct {
	clusters        clusterDataStore.DataStore
	serviceaccounts datastore.DataStore
	reconcileStore  reconciliation.Store
}

func (s *pipelineImpl) Reconcile(clusterID string) error {
	defer s.reconcileStore.Close()

	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.serviceaccounts.Search(query)
	if err != nil {
		return err
	}

	idsToDelete := search.ResultsToIDSet(results).Difference(s.reconcileStore.GetSet()).AsSlice()
	if len(idsToDelete) > 0 {
		log.Infof("Deleting service accounts %+v as a part of reconciliation", idsToDelete)
	}
	errList := errorhelpers.NewErrorList("Service Account reconciliation")
	for _, id := range idsToDelete {
		errList.AddError(s.runRemovePipeline(central.ResourceAction_REMOVE_RESOURCE, &storage.ServiceAccount{Id: id}))
	}
	return errList.ToError()
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetServiceAccount() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	if !features.K8sRBAC.Enabled() {
		return nil
	}

	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ServiceAccount)

	event := msg.GetEvent()
	sa := event.GetServiceAccount()
	sa.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event.GetAction(), sa)
	default:
		s.reconcileStore.Add(event.GetId())
		return s.runGeneralPipeline(event.GetAction(), sa)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action central.ResourceAction, event *storage.ServiceAccount) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistServiceAccount(action, event); err != nil {
		return err
	}

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action central.ResourceAction, sa *storage.ServiceAccount) error {
	if err := s.validateInput(sa); err != nil {
		return err
	}

	if err := s.enrichCluster(sa); err != nil {
		return err
	}

	if err := s.persistServiceAccount(action, sa); err != nil {
		return err
	}

	return nil
}

func (s *pipelineImpl) validateInput(sa *storage.ServiceAccount) error {
	// validate input.
	if sa == nil {
		return fmt.Errorf("service account must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(sa *storage.ServiceAccount) error {
	sa.ClusterName = ""

	cluster, clusterExists, err := s.clusters.GetCluster(sa.GetClusterId())
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
		return err
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", sa.GetClusterId())
		return err
	default:
		sa.ClusterName = cluster.GetName()
	}
	return nil
}

func (s *pipelineImpl) persistServiceAccount(action central.ResourceAction, sa *storage.ServiceAccount) error {
	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE:
		return s.serviceaccounts.UpsertServiceAccount(sa)
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.serviceaccounts.RemoveServiceAccount(sa.GetId())
	default:
		return fmt.Errorf("Event action '%s' for service account does not exist", action)
	}
}

func (s *pipelineImpl) OnFinish() {}
