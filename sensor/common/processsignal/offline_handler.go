package processsignal

import (
	"github.com/stackrox/rox/sensor/common/message"
	"golang.org/x/net/context"
)

type drofOfflineHandler struct{}

func (d *drofOfflineHandler) HandleOffline(m *message.ExpiringMessage) {
	log.Debugf("Handling ProcessSignal in Offline mode: %s", m.String())
	canceledCtx, cancel := context.WithCancel(m.Context)
	cancel()
	m.Context = canceledCtx
}
