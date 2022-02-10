package handler

import "time"

// handlerOpts represents the options for a scannerdefinitions http.Handler.
type handlerOpts struct {
	// offlineVulnDefsDir is the directory in which persisted vulnerability definitions should be written.
	// It is assumed the directory already exists.
	// Default: /var/lib/stackrox/scannerdefinitions
	offlineVulnDefsDir string

	// The following are options which are only respected in online-mode.

	// cleanupInterval sets the interval for cleaning up updaters.
	cleanupInterval *time.Duration
	// cleanupAge sets the age after which an updater should be cleaned.
	cleanupAge *time.Duration
}
