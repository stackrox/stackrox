package serviceaccounts

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), deploymentDataStore.Singleton(), datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline for service accounts
func NewPipeline(clusters clusterDataStore.DataStore, deployments deploymentDataStore.DataStore, serviceaccounts datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusters:             clusters,
		deployments:          deployments,
		serviceaccounts:      serviceaccounts,
		riskReprocessor:      reprocessor.Singleton(),
		reconciliationSignal: concurrency.NewSignal(),
	}
}

type pipelineImpl struct {
	clusters        clusterDataStore.DataStore
	deployments     deploymentDataStore.DataStore
	serviceaccounts datastore.DataStore
	riskReprocessor reprocessor.Loop

	reconciliationSignal concurrency.Signal
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	// Signal before running with reconciliation to avoid a potential race with Run() calls
	// Calling this here will cause some duplicate risk reprocessing, but overall should be
	// significantly less than without the reconciliation signal
	s.reconciliationSignal.Signal()
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.serviceaccounts.Search(ctx, query)
	if err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_ServiceAccount)(nil))
	return reconciliation.Perform(store, search.ResultsToIDSet(results), "service accounts", func(id string) error {
		return s.runRemovePipeline(ctx, &storage.ServiceAccount{Id: id})
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetServiceAccount() != nil
}

var (
	lock sync.Mutex
	m    = make(map[string]*storage.ServiceAccount)
)

func checkDiff(sa *storage.ServiceAccount) {
	lock.Lock()
	defer lock.Unlock()

	old, ok := m[sa.GetId()]
	if !ok {
		m[sa.GetId()] = sa
		return
	}
	m[sa.GetId()] = sa
	if proto.Equal(old, sa) {
		log.Infof("Equal %+v %+v", old, sa)
	} else {
		log.Infof("Not equal %+v %+v", old, sa)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ServiceAccount)

	event := msg.GetEvent()
	sa := event.GetServiceAccount()
	sa.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(ctx, sa)
	default:
		checkDiff(sa)
		return s.runGeneralPipeline(ctx, event.GetAction(), sa)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, event *storage.ServiceAccount) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistServiceAccount(ctx, central.ResourceAction_REMOVE_RESOURCE, event); err != nil {
		return err
	}

	return nil
}

func (s *pipelineImpl) reprocessRisk(ctx context.Context, sa *storage.ServiceAccount) error {
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, sa.ClusterId).
		AddExactMatches(search.Namespace, sa.Namespace).
		AddExactMatches(search.ServiceAccountName, sa.Name).ProtoQuery()

	results, err := s.deployments.Search(ctx, q)
	if err != nil {
		log.Errorf("error searching for deployments with service account %q", sa.GetName())
		return err
	}
	deploymentIDs := search.ResultsToIDs(results)
	// Reprocess risk
	s.riskReprocessor.ReprocessRiskForDeployments(deploymentIDs...)
	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, action central.ResourceAction, sa *storage.ServiceAccount) error {
	if err := s.validateInput(sa); err != nil {
		return err
	}

	if err := s.enrichCluster(ctx, sa); err != nil {
		return err
	}

	if err := s.persistServiceAccount(ctx, action, sa); err != nil {
		return err
	}

	// If we have completely reconciliation then reevaluate risk on every service account event
	if s.reconciliationSignal.IsDone() {
		return s.reprocessRisk(ctx, sa)
	}
	return nil
}

func (s *pipelineImpl) validateInput(sa *storage.ServiceAccount) error {
	// validate input.
	if sa == nil {
		return errors.New("service account must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(ctx context.Context, sa *storage.ServiceAccount) error {
	sa.ClusterName = ""

	clusterName, clusterExists, err := s.clusters.GetClusterName(ctx, sa.GetClusterId())
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
		return err
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", sa.GetClusterId())
		return err
	default:
		sa.ClusterName = clusterName
	}
	return nil
}

func (s *pipelineImpl) persistServiceAccount(ctx context.Context, action central.ResourceAction, sa *storage.ServiceAccount) error {
	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		return s.serviceaccounts.UpsertServiceAccount(ctx, sa)
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.serviceaccounts.RemoveServiceAccount(ctx, sa.GetId())
	default:
		return fmt.Errorf("Event action '%s' for service account does not exist", action)
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
