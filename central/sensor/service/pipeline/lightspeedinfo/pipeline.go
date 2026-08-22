package lightspeedinfo

import (
	"context"

	"github.com/stackrox/rox/central/lightspeed/store"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// LightspeedStore is the interface for storing Lightspeed info.
type LightspeedStore interface {
	UpdateInfo(clusterID string, info *central.LightspeedInfo)
}

type pipelineImpl struct {
	store LightspeedStore
}

// NewPipeline returns a new instance of the Lightspeed info pipeline.
func NewPipeline(store LightspeedStore) pipeline.Fragment {
	return &pipelineImpl{store: store}
}

// GetPipeline returns a new pipeline using the singleton store.
func GetPipeline() pipeline.Fragment {
	return NewPipeline(store.Singleton())
}

func (p *pipelineImpl) OnFinish(_ string) {}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetLightspeedInfo() != nil
}

// Run processes the incoming Lightspeed info and stores it.
func (p *pipelineImpl) Run(_ context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	p.store.UpdateInfo(clusterID, msg.GetLightspeedInfo())
	return nil
}

func (p *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}
