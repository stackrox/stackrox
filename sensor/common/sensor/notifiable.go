package sensor

import "github.com/stackrox/rox/sensor/common"

// OfflineAware is meant to replace common.Notifiable for non-components, so that a pkg unrelated to Sensor
// is not forced to import sensor code.
type OfflineAware interface {
	GoOnline()
	GoOffline()
}

// WrapNotifiable makes OfflineAware struct implement the Notifiable interface
func WrapNotifiable(oa OfflineAware, name string) common.Notifiable {
	return &notifiableImpl{
		name: name,
		oa:   oa,
	}
}

type notifiableImpl struct {
	name string
	oa   OfflineAware
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
