package admissioncontroller

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

var (
	instance *alertHandlerImpl
	once     sync.Once
)

func newAlertHandler() *alertHandlerImpl {
	alertHandler := &alertHandlerImpl{
		stopSig:      concurrency.NewSignal(),
		output:       make(chan *message.ExpiringMessage),
		centralReady: concurrency.NewSignal(),
	}
	common.RegisterStateReporter(alertHandlerComponentName, alertHandler.State)
	return alertHandler
}

// AlertHandlerSingleton returns the singleton instance for the admission controller alert handler handler.
func AlertHandlerSingleton() AlertHandler {
	once.Do(func() {
		instance = newAlertHandler()
	})
	return instance
}
