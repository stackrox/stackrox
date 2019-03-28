package manager

import v1 "github.com/stackrox/rox/generated/api/v1"

// LicenseEventListener defines a listener interface for license events.
type LicenseEventListener interface {
	// OnActiveLicenseChanged gets called whenever the active license changes (including deactivation of
	// an existing license, in which case `newLicense` will be `nil`). To ensure strict in-order guarantees, this
	// function is invoked *synchronously*, hence its implementation should spawn a goroutine for tasks that might
	// block for a longer period of time.
	OnActiveLicenseChanged(newLicense, oldLicense *v1.LicenseInfo)
}

//go:generate mockgen-wrapper LicenseEventListener
