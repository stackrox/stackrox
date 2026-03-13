package common

import (
	"time"

	"github.com/cenkalti/backoff/v5"
)

const (
	// BackoffResetThreshold indicates how long a connection must last in order to reset the exponential backoff timer.
	BackoffResetThreshold = 10 * time.Second
)

// NewBackOffForSensorConn return an exponentialBackOff for sensor connection.
func NewBackOffForSensorConn() *backoff.ExponentialBackOff {
	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = 10 * time.Second
	eb.MaxInterval = 1 * time.Minute
	eb.Reset()

	return eb
}
