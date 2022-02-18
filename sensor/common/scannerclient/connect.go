package scannerclient

import (
	"time"

	"github.com/cenkalti/backoff/v3"
)

func waitUntilScannerIsReady() {
	exponential := backoff.NewExponentialBackOff()
	exponential.MaxElapsedTime = 5 * time.Minute
	exponential.MaxInterval = 32 * time.Second

	err := backoff.RetryNotify(func() error {
		return s.pollMetadata()
	}, exponential, func(err error, d time.Duration) {
		log.Infof("Check Central status failed: %s. Retrying after %s...", err, d.Round(time.Millisecond))
	})
	if err != nil {
		s.stoppedSig.SignalWithErrorWrapf(err, "checking central status failed after %s", exponential.GetElapsedTime())
	}
}
