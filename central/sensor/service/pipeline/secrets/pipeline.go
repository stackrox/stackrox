package secrets

import (
	"fmt"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/secret/datastore"
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

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline for secrets
func NewPipeline(clusters clusterDataStore.DataStore, secrets datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusters:       clusters,
		secrets:        secrets,
		reconcileStore: reconciliation.NewStore(),
	}
}

type pipelineImpl struct {
	clusters       clusterDataStore.DataStore
	secrets        datastore.DataStore
	reconcileStore reconciliation.Store
}

func (s *pipelineImpl) Reconcile(clusterID string) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.secrets.Search(query)
	if err != nil {
		return err
	}

	return reconciliation.Perform(s.reconcileStore, search.ResultsToIDSet(results), "secrets", func(id string) error {
		return s.runRemovePipeline(central.ResourceAction_REMOVE_RESOURCE, &storage.Secret{Id: id})
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetSecret() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Secret)

	event := msg.GetEvent()
	secret := event.GetSecret()
	secret.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event.GetAction(), secret)
	default:
		s.reconcileStore.Add(event.GetId())
		return s.runGeneralPipeline(event.GetAction(), secret)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action central.ResourceAction, event *storage.Secret) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistSecret(action, event); err != nil {
		return err
	}

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action central.ResourceAction, secret *storage.Secret) error {
	if err := s.validateInput(secret); err != nil {
		return err
	}

	if err := s.enrichCluster(secret); err != nil {
		return err
	}

	if err := s.persistSecret(action, secret); err != nil {
		return err
	}

	return nil
}

func (s *pipelineImpl) validateInput(secret *storage.Secret) error {
	// validate input.
	if secret == nil {
		return fmt.Errorf("secret must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(secret *storage.Secret) error {
	secret.ClusterName = ""

	cluster, clusterExists, err := s.clusters.GetCluster(secret.GetClusterId())
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", secret.GetClusterId())
	default:
		secret.ClusterName = cluster.GetName()
	}
	return nil
}

func (s *pipelineImpl) persistSecret(action central.ResourceAction, secret *storage.Secret) error {
	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE:
		return s.secrets.UpsertSecret(secret)
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.secrets.RemoveSecret(secret.GetId())
	default:
		return fmt.Errorf("Event action '%s' for secret does not exist", action)
	}
}

func (s *pipelineImpl) OnFinish(clusterID string) {}
