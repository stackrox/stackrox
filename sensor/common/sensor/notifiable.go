package sensor

import "github.com/stackrox/rox/sensor/common"

// offlineAware is meant to replace common.Notifiable for non-components, so that a pkg unrelated to Sensor
// is not forced to import sensor code.
type offlineAware interface {
	GoOnline()
	GoOffline()
}

// wrapNotifiable makes offlineAware struct implement the Notifiable interface
func wrapNotifiable(oa offlineAware, name string) common.Notifiable {
	return &notifiableImpl{
		name: name,
		oa:   oa,
	}
}

type notifiableImpl struct {
	name string
	oa   offlineAware
}

func (ni *notifiableImpl) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, ni.name))
	switch e {
	case common.SensorComponentEventCentralReachable:
		ni.oa.GoOnline()
	case common.SensorComponentEventOfflineMode:
		ni.oa.GoOffline()
	}
}
