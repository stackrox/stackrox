package scannerclient

import (
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

var (
	_ common.Notifiable = (*resetNotifiable)(nil)

	notifiable     *resetNotifiable
	notifiableOnce sync.Once
)

// ResetNotifiable returns a notifiable that resets
// the gRPC client singleton when notified.
func ResetNotifiable() common.Notifiable {
	notifiableOnce.Do(func() {
		notifiable = &resetNotifiable{}
	})

	return notifiable
}

// resetNotifiable will reset the scanner client singleton when notified.
// This allows the scanner client to be recreated on next retrieval, allowing
// for re-evaluation, for example, of central capabilities.
type resetNotifiable struct {
	common.Notifiable
}

func (r *resetNotifiable) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		resetGRPCClient()
		log.Debug("Reset scanner client")
	}
}
