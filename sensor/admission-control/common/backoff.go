package common

import (
	"time"

	"github.com/cenkalti/backoff/v3"
)

const (
	// BackoffResetThreshold indicates how long a connection must last in order to reset the exponential backoff timer.
	BackoffResetThreshold = 10 * time.Second
)

// NewBackOffForSensorConn return an exponentialBackOff for sensor connection.
func NewBackOffForSensorConn() *backoff.ExponentialBackOff {
	eb := backoff.NewExponentialBackOff()
	eb.MaxInterval = 1 * time.Minute
	eb.InitialInterval = 10 * time.Second
	eb.MaxElapsedTime = 365 * 24 * time.Hour
	eb.Reset()

	return eb
}
