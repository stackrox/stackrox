package lightspeedquery

import (
	"context"

	"github.com/stackrox/rox/central/lightspeed/broker"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// QueryBroker is the interface that will be notified when a Lightspeed query response arrives.
type QueryBroker interface {
	NotifyResponseReceived(resp *central.LightspeedQueryResponse)
}

type pipelineImpl struct {
	broker QueryBroker
}

// NewPipeline returns a new instance of the Lightspeed query pipeline.
func NewPipeline(broker QueryBroker) pipeline.Fragment {
	return &pipelineImpl{broker: broker}
}

// GetPipeline returns a new pipeline using the singleton broker.
func GetPipeline() pipeline.Fragment {
	return NewPipeline(broker.Singleton())
}

func (p *pipelineImpl) OnFinish(_ string) {}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetLightspeedQueryResponse() != nil
}

// Run processes the incoming Lightspeed query response and routes it to the broker.
func (p *pipelineImpl) Run(_ context.Context, _ string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	p.broker.NotifyResponseReceived(msg.GetLightspeedQueryResponse())
	return nil
}

func (p *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}
