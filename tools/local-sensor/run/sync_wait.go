package run

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
)

// registerSyncWait installs an OnMessage handler that signals when Sensor sends ResourcesSynced.
// Works with SkipCentralOutput (recording off): callbacks still run via invokeMessageCallback.
func registerSyncWait(fakeCentral *centralDebug.FakeService, verboseLog func(*central.MsgFromSensor)) func(context.Context) error {
	syncSig := concurrency.NewSignal()

	fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
		if msg.GetEvent().GetSynced() != nil {
			syncSig.Signal()
		}
		if verboseLog != nil {
			verboseLog(msg)
		}
	})

	return func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-syncSig.Done():
			return nil
		}
	}
}
