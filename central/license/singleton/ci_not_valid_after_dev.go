// +build !release

package singleton

import (
	"time"
)

const (
	// This time is set really high because we may need to run a release in CI for up to 2 years after it was first
	// released, to just upgrades from that release.
	ciSigningKeyLatestNotValidAfterOffset = 2 * 365 * 24 * time.Hour
)
