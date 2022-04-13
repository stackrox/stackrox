package transfer

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/concurrency"
)

func watchdog(errSig *concurrency.ErrorSignal, earliestDeadline time.Time, lastActivity func() time.Time, idleTimeout time.Duration) {
	nextT := time.After(time.Until(earliestDeadline))

	for !errSig.IsDone() {
		select {
		case <-errSig.Done():
		case <-nextT:
			if time.Since(lastActivity()) > idleTimeout {
				errSig.SignalWithError(errors.Errorf("no I/O activity for the last %v", idleTimeout))
				return
			}
			nextT = time.After(idleTimeout + 1*time.Second)
		}
	}
}
