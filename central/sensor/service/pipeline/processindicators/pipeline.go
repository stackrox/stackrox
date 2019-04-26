package processindicators

import (
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	countMetrics "github.com/stackrox/rox/central/metrics"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processwhitelist"
	whitelistDataStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(processDataStore.Singleton(), whitelistDataStore.Singleton(), datastore.Singleton(), lifecycle.SingletonManager())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(indicators processDataStore.DataStore, whitelists whitelistDataStore.DataStore, deployments datastore.DataStore, manager lifecycle.Manager) pipeline.Fragment {
	return &pipelineImpl{
		indicators:  indicators,
		whitelists:  whitelists,
		manager:     manager,
		deployments: deployments,
	}
}

type pipelineImpl struct {
	indicators  processDataStore.DataStore
	whitelists  whitelistDataStore.DataStore
	deployments datastore.DataStore
	manager     lifecycle.Manager
}

func (s *pipelineImpl) Reconcile(clusterID string) error {
	// Nothing to reconcile
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetProcessIndicator() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(_ string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ProcessIndicator)

	event := msg.GetEvent()
	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.indicators.RemoveProcessIndicator(event.GetProcessIndicator().GetId())
	default:
		return s.process(event.GetProcessIndicator(), injector)
	}
}

func (s *pipelineImpl) CheckWhitelist(indicator *storage.ProcessIndicator) error {
	key := &storage.ProcessWhitelistKey{
		DeploymentId:  indicator.DeploymentId,
		ContainerName: indicator.ContainerName,
	}

	// TODO joseph what to do if whitelist doesn't exist?  Always create for now?
	whitelist, err := s.whitelists.GetProcessWhitelist(key)
	if err != nil {
		return err
	}

	insertableElement := &storage.WhitelistItem{Item: &storage.WhitelistItem_ProcessName{ProcessName: indicator.GetSignal().GetExecFilePath()}}
	if whitelist == nil {
		_, err := s.whitelists.UpsertProcessWhitelist(key, []*storage.WhitelistItem{insertableElement}, true)
		return err
	}

	for _, element := range whitelist.GetElements() {
		if element.GetElement().GetProcessName() == insertableElement.GetProcessName() {
			return nil
		}
	}
	if processwhitelist.IsLocked(whitelist.GetUserLockedTimestamp()) {
		// TODO joseph create an alert
		return nil
	}
	if processwhitelist.IsLocked(whitelist.GetStackRoxLockedTimestamp()) {
		// TODO joseph create risk
		return nil
	}
	_, err = s.whitelists.UpdateProcessWhitelistElements(key, []*storage.WhitelistItem{insertableElement}, nil, true)
	return err
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) process(indicator *storage.ProcessIndicator, injector common.MessageInjector) error {
	if features.ProcessWhitelist.Enabled() {
		err := s.CheckWhitelist(indicator)
		if err != nil {
			log.Error(err)
		}
	}
	return s.manager.IndicatorAdded(indicator, injector)
}

func (s *pipelineImpl) OnFinish(clusterID string) {}
