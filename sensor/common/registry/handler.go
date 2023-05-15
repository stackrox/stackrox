package registry

import (
	"errors"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/common"
)

// DelegatedRegistryConfigHandler is responsible for processing delegated
// registry config updates from central
type DelegatedRegistryConfigHandler interface {
	common.SensorComponent
}

type delegatedRegistryConfigImpl struct {
	registryStore *Store
	stopSig       concurrency.Signal
}

// NewDelegatedRegistryConfigHandler returns a new instance of DelegatedRegistryConfigHandler
func NewDelegatedRegistryConfigHandler(registryStore *Store) DelegatedRegistryConfigHandler {
	return &delegatedRegistryConfigImpl{
		registryStore: registryStore,
		stopSig:       concurrency.NewSignal(),
	}
}

func (d *delegatedRegistryConfigImpl) Capabilities() []centralsensor.SensorCapability {
	if !env.LocalImageScanningEnabled.BooleanSetting() {
		// do not advertise the capability if local scanning is disabled
		return nil
	}

	return []centralsensor.SensorCapability{centralsensor.DelegatedRegistryCap}
}

func (d *delegatedRegistryConfigImpl) Notify(_ common.SensorComponentEvent) {}

func (d *delegatedRegistryConfigImpl) ProcessMessage(msg *central.MsgToSensor) error {
	if !env.LocalImageScanningEnabled.BooleanSetting() {
		// ignore all messages if local scanning is disabled
		return nil
	}

	config := msg.GetUpdatedDelegatedRegistryConfig()
	if config == nil {
		return nil
	}

	select {
	case <-d.stopSig.Done():
		return errors.New("could not process updated delegated registry config")
	default:
		d.registryStore.SetDelegatedRegistryConfig(config)
		log.Debugf("Stored updated delegated registry config: %q", config)
	}

	return nil
}

func (d *delegatedRegistryConfigImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

func (d *delegatedRegistryConfigImpl) Start() error {
	return nil
}

func (d *delegatedRegistryConfigImpl) Stop(_ error) {
	d.stopSig.Signal()
}
