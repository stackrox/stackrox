package reprocessor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/store"
)

// Handler handles request to reprocess deployment (sent by Central).
type Handler interface {
	common.SensorComponent
}

// NewHandler returns a new instance of a deployment reprocessor.
func NewHandler(deploymentStore store.DeploymentStore) Handler {
	return &handlerImpl{
		deploymentStore: deploymentStore,
	}
}

type handlerImpl struct {
	deploymentStore store.DeploymentStore
	detector        detector.Detector
	stopSig         concurrency.ErrorSignal
}

func (h *handlerImpl) Start() error {
	return nil
}

func (h *handlerImpl) Stop(err error) {
	h.stopSig.SignalWithError(err)
}

func (h *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	// A new sensor capability to reprocess deployment has not been added. In case of mismatched upgrades,
	// the re-processing is discarded, which is fine.
	return nil
}

func (h *handlerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	request := msg.GetReprocessDeployment()
	if request == nil {
		return nil
	}
	deploymentID := request.GetDeploymentId()
	select {
	case <-h.stopSig.Done():
		return errors.Wrapf(h.stopSig.Err(), "could not fulfill re-process deployment %q request", request.GetDeploymentId())
	default:
		deployment := h.deploymentStore.Get(deploymentID)
		// The deployment affected by the vulnerability request may have been removed.
		if deployment == nil {
			return nil
		}
		go h.detector.ReprocessDeployment(deployment)
		return nil
	}
}

func (h *handlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}
