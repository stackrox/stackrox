package handler

import "time"

// handlerOpts represents the options for a scannerdefinitions http.Handler.
type handlerOpts struct {
	// The following are options which are only respected in online-mode.
	// cleanupInterval sets the interval for cleaning up updaters.
	cleanupInterval *time.Duration

	// cleanupAge sets the age after which an updater should be cleaned.
	cleanupAge *time.Duration
}
