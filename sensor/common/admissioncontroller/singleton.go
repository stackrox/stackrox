package admissioncontroller

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
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
