package secrets

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
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
	return NewPipeline(clusterDataStore.Singleton(), datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline for secrets
func NewPipeline(clusters clusterDataStore.DataStore, secrets datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusters: clusters,
		secrets:  secrets,
	}
}

type pipelineImpl struct {
	clusters clusterDataStore.DataStore
	secrets  datastore.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.secrets.Search(ctx, query)
	if err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_Secret)(nil))
	return reconciliation.Perform(store, search.ResultsToIDSet(results), "secrets", func(id string) error {
		return s.runRemovePipeline(ctx, central.ResourceAction_REMOVE_RESOURCE, &storage.Secret{Id: id})
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetSecret() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.Secret)

	event := msg.GetEvent()
	secret := event.GetSecret()
	secret.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(ctx, event.GetAction(), secret)
	default:
		return s.runGeneralPipeline(ctx, event.GetAction(), secret)
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, action central.ResourceAction, event *storage.Secret) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the deployment from persistence depending on the event action.
	if err := s.persistSecret(ctx, action, event); err != nil {
		return err
	}

	return nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, action central.ResourceAction, secret *storage.Secret) error {
	if err := s.validateInput(secret); err != nil {
		return err
	}

	if err := s.enrichCluster(ctx, secret); err != nil {
		return err
	}

	if err := s.persistSecret(ctx, action, secret); err != nil {
		return err
	}

	return nil
}

func (s *pipelineImpl) validateInput(secret *storage.Secret) error {
	// validate input.
	if secret == nil {
		return errors.New("secret must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(ctx context.Context, secret *storage.Secret) error {
	secret.ClusterName = ""

	clusterName, clusterExists, err := s.clusters.GetClusterName(ctx, secret.GetClusterId())
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", secret.GetClusterId())
	default:
		secret.ClusterName = clusterName
	}
	return nil
}

var (
	lock sync.Mutex
	m    = make(map[string]*storage.Secret)
)

func checkDiff(secret *storage.Secret) {
	lock.Lock()
	defer lock.Unlock()

	old, ok := m[secret.GetId()]
	if !ok {
		m[secret.GetId()] = secret
		return
	}
	m[secret.GetId()] = secret
	if proto.Equal(old, secret) {
		log.Infof("Equal %+v %+v", old, secret)
	} else {
		log.Infof("Not equal %+v %+v", old, secret)
	}
}

func (s *pipelineImpl) persistSecret(ctx context.Context, action central.ResourceAction, secret *storage.Secret) error {
	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		checkDiff(secret)
		return s.secrets.UpsertSecret(ctx, secret)
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.secrets.RemoveSecret(ctx, secret.GetId())
	default:
		return fmt.Errorf("Event action '%s' for secret does not exist", action)
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
