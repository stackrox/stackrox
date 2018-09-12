package secrets

import (
	"fmt"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// NewPipeline returns a new instance of Pipeline for secrets
func NewPipeline(clusters clusterDataStore.DataStore, secrets datastore.DataStore) pipeline.Pipeline {
	return &pipelineImpl{
		clusters: clusters,
		secrets:  secrets,
	}
}

type pipelineImpl struct {
	clusters clusterDataStore.DataStore
	secrets  datastore.DataStore
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(event *v1.SensorEvent) (*v1.SensorEnforcement, error) {
	secret := event.GetSecret()
	secret.ClusterId = event.GetClusterId()
	switch event.GetAction() {
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(event.GetAction(), secret)
	default:
		return s.runGeneralPipeline(event.GetAction(), secret)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(action v1.ResourceAction, event *v1.Secret) (*v1.SensorEnforcement, error) {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return nil, err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistSecret(action, event); err != nil {
		return nil, err
	}

	return nil, nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(action v1.ResourceAction, secret *v1.Secret) (*v1.SensorEnforcement, error) {
	if err := s.validateInput(secret); err != nil {
		return nil, err
	}

	if err := s.enrichCluster(secret); err != nil {
		return nil, err
	}

	if err := s.persistSecret(action, secret); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *pipelineImpl) validateInput(secret *v1.Secret) error {
	// validate input.
	if secret == nil {
		return fmt.Errorf("secret must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(secret *v1.Secret) error {
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

func (s *pipelineImpl) persistSecret(action v1.ResourceAction, secret *v1.Secret) error {
	switch action {
	case v1.ResourceAction_CREATE_RESOURCE, v1.ResourceAction_UPDATE_RESOURCE:
		return s.secrets.UpsertSecret(secret)
	case v1.ResourceAction_REMOVE_RESOURCE:
		return s.secrets.RemoveSecret(secret.GetId())
	default:
		return fmt.Errorf("Event action '%s' for secret does not exist", action)
	}
}
