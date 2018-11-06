package eventstream

import "github.com/stackrox/rox/generated/api/v1"

// SensorEventStream is a stripped-down version of the SensorEventService RecordEvents stream.
type SensorEventStream interface {
	Send(event *v1.SensorEvent) error
	SendRaw(event *v1.SensorEvent, raw []byte) error
}
