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

// ResetNotifiableSingleton returns a notifiable that resets
// the gRPC client singleton when notified.
func ResetNotifiableSingleton() common.Notifiable {
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
	switch e {
	case common.SensorComponentEventCentralReachable:
		log.Debugf("Resetting scanner client")
		resetGRPCClient()
	}
}
