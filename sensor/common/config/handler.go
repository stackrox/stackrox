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

// Handler is responsible for processing dynamic config updates from central and, for Helm-managed clusters, to provide
// access to the cluster's configuration.
type Handler interface {
	GetConfig() *storage.DynamicClusterConfig
	GetHelmManagedConfig() *central.HelmManagedConfigInit
	GetDeploymentIdentification() *storage.SensorDeploymentIdentification

	common.SensorComponent
}

// NewCommandHandler returns a new instance of a Handler.
func NewCommandHandler(admCtrlSettingsMgr admissioncontroller.SettingsManager, deploymentIdentification *storage.SensorDeploymentIdentification, helmManagedConfig *central.HelmManagedConfigInit) Handler {
	return &configHandlerImpl{
		stopC:                    concurrency.NewErrorSignal(),
		admCtrlSettingsMgr:       admCtrlSettingsMgr,
		helmManagedConfig:        helmManagedConfig,
		deploymentIdentification: deploymentIdentification,
	}
}

type configHandlerImpl struct {
	deploymentIdentification *storage.SensorDeploymentIdentification
	helmManagedConfig        *central.HelmManagedConfigInit

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

func (c *configHandlerImpl) GetHelmManagedConfig() *central.HelmManagedConfigInit {
	return c.helmManagedConfig
}

func (c *configHandlerImpl) GetDeploymentIdentification() *storage.SensorDeploymentIdentification {
	return c.deploymentIdentification
}
