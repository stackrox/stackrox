package reposcan

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/baseimage/broker"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
)

var _ pipeline.Fragment = (*pipelineImpl)(nil)

// RepoScanBroker is the interface for receiving repository scan responses from Sensor.
type RepoScanBroker interface {
	OnScanResponse(clusterID string, msg *central.RepoScanResponse)
	OnClusterDisconnect(clusterID string)
}

type pipelineImpl struct {
	broker RepoScanBroker
}

// NewPipeline returns a new instance of the RepoScan Pipeline.
func NewPipeline(broker RepoScanBroker) pipeline.Fragment {
	return &pipelineImpl{broker: broker}
}

// GetPipeline returns a new pipeline.
func GetPipeline() pipeline.Fragment {
	return NewPipeline(broker.Singleton())
}

func (p *pipelineImpl) OnFinish(clusterID string) {
	p.broker.OnClusterDisconnect(clusterID)
}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetRepoScanResponse() != nil
}

// Run processes a RepoScanResponse message from Sensor.
func (p *pipelineImpl) Run(_ context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	r := msg.GetRepoScanResponse()
	if r == nil {
		return fmt.Errorf("reposcan request is nil: cluster id: %q", clusterID)
	}
	if r.GetRequestId() == "" {
		return fmt.Errorf("reposcan request id is empty: cluster id: %q", clusterID)
	}
	p.broker.OnScanResponse(clusterID, r)
	return nil
}

func (p *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}
