package admissioncontroller

import (
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	instance *alertHandlerImpl
	once     sync.Once
)

func newAlertHandler() *alertHandlerImpl {
	return &alertHandlerImpl{
		stopSig: concurrency.NewSignal(),
		output:  make(chan *central.MsgFromSensor),
	}
}

// AlertHandlerSingleton returns the singleton instance for the admission controller alert handler handler.
func AlertHandlerSingleton() AlertHandler {
	once.Do(func() {
		instance = newAlertHandler()
	})
	return instance
}
