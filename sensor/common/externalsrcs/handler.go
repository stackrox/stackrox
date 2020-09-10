package externalsrcs

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

var (
	log = logging.LoggerForModule()
)

// Handler forwards the external network entities received from Central to Collectors.
type Handler interface {
	common.SensorComponent
}

// NewHandler returns a new ready-to-use updater.
func NewHandler() common.SensorComponent {
	return &handlerImpl{
		stopSig: concurrency.NewSignal(),
	}
}

type handlerImpl struct {
	stopSig concurrency.Signal

	entities         []*storage.NetworkEntityInfo
	lastRequestSeqID int64

	lock sync.Mutex
}

func (h *handlerImpl) Start() error {
	return nil
}

func (h *handlerImpl) Stop(_ error) {
	h.stopSig.Signal()
}

func (h *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.NetworkGraphExternalSrcsCap}
}

func (h *handlerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	request := msg.GetPushNetworkEntitiesRequest()
	if request == nil {
		return nil
	}
	select {
	case <-h.stopSig.Done():
		return errors.New("could not process external network entities request")
	default:
		h.lock.Lock()
		defer h.lock.Unlock()

		if request.GetSeqID() < h.lastRequestSeqID {
			return nil
		}

		h.entities = request.GetEntities()
		h.lastRequestSeqID = request.GetSeqID()
		// TODO(ROX-5465): Push to collector.
		return nil
	}
}

func (h *handlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}
