package config

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
)

var (
	log = logging.LoggerForModule()
)

// Handler executes the input scrape commands, and reconciles scrapes with input ComplianceReturns,
// outputing the ScrapeUpdates we expect to be sent back to central.
type Handler interface {
	GetConfig() *storage.DynamicClusterConfig
	common.SensorComponent
}

// NewCommandHandler returns a new instance of a Handler using the input image and Orchestrator.
func NewCommandHandler(admCtrlSettingsMgr admissioncontroller.SettingsManager) Handler {
	return &configHandlerImpl{
		stopC:              concurrency.NewErrorSignal(),
		admCtrlSettingsMgr: admCtrlSettingsMgr,
	}
}

type configHandlerImpl struct {
	config *storage.DynamicClusterConfig
	lock   sync.RWMutex

	admCtrlSettingsMgr admissioncontroller.SettingsManager

	stopC concurrency.ErrorSignal
}

func (c *configHandlerImpl) Start() error {
	return nil
}

func (c *configHandlerImpl) Stop(_ error) {
	c.stopC.Signal()
}

func (c *configHandlerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (c *configHandlerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

func (c *configHandlerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	config := msg.GetClusterConfig()
	if config == nil {
		return nil
	}

	select {
	case <-c.stopC.Done():
		return errors.New("could not process new cluster config")
	default:
		log.Infof("Received configuration from Central: %s", protoutils.NewWrapper(config))
		c.lock.Lock()
		defer c.lock.Unlock()
		c.config = config.Config
		if c.admCtrlSettingsMgr != nil {
			c.admCtrlSettingsMgr.UpdateConfig(config.GetConfig())
		}
		return nil
	}
}

func (c *configHandlerImpl) GetConfig() *storage.DynamicClusterConfig {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.config
}
