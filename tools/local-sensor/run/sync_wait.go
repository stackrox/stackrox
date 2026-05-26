package run

import (
	"context"
	"time"

	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
)

const syncPollInterval = 500 * time.Millisecond

func waitForSyncEvent(ctx context.Context, fakeCentral *centralDebug.FakeService) error {
	ticker := time.NewTicker(syncPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for _, msg := range fakeCentral.GetAllMessages() {
				if msg.GetEvent().GetSynced() != nil {
					return nil
				}
			}
		}
	}
}

func newWaitInitialSync(fakeCentral *centralDebug.FakeService) func(context.Context) error {
	if fakeCentral == nil {
		return func(context.Context) error { return nil }
	}
	return func(ctx context.Context) error {
		return waitForSyncEvent(ctx, fakeCentral)
	}
}
