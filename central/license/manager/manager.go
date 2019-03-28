package manager

import (
	"github.com/stackrox/rox/central/license/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/license/validator"
)

// LicenseManager is responsible for managing product licenses.
type LicenseManager interface {

	// Initialize starts the license manager and returns the active license, if any. The listener is registered
	// synchronously and will deliver any license event *after* the selection of an initially active license.
	Initialize(listener LicenseEventListener) (*v1.License, error)
	Stop() concurrency.Waitable

	GetActiveLicense() *v1.License
	GetAllLicenses() []*v1.LicenseInfo

	AddLicenseKey(licenseKey string) (*v1.LicenseInfo, error)
	SelectLicense(licenseID string) (*v1.LicenseInfo, error)
}

// New creates and returns a new license manager, using the given license key store and validator.
func New(store store.Store, validator validator.Validator) LicenseManager {
	return newManager(store, validator)
}
