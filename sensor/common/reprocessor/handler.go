package reprocessor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/detector"
)

var (
	log = logging.LoggerForModule()
)

// Handler handles request to reprocess deployment (sent by Central).
type Handler interface {
	common.SensorComponent
}

// NewHandler returns a new instance of a deployment reprocessor.
func NewHandler(detector detector.Detector) Handler {
	return &handlerImpl{
		detector: detector,
		stopSig:  concurrency.NewErrorSignal(),
	}
}

type handlerImpl struct {
	detector detector.Detector
	stopSig  concurrency.ErrorSignal
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
	req := msg.GetReprocessDeployment()
	if req == nil {
		return nil
	}
	log.Debug("Received request to reprocess deployments from Central")

	select {
	case <-h.stopSig.Done():
		return errors.Wrap(h.stopSig.Err(), "could not fulfill re-process deployment(s) request")
	default:
		go h.detector.ReprocessDeployments(req.GetDeploymentIds()...)
	}
	return nil
}

func (h *handlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}
