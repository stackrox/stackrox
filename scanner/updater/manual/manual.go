// Package manual provides a custom updater for vulnerability scanner.
// This updater allows manual input of vulnerability data.
package manual

import (
	"context"
	"io"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
)

// Factory is the UpdaterSetFactory exposed by this package.
// All configuration is done on the returned updaters.
type Factory struct {
}

// UpdaterSet creates a new updater set with the provided vulnerability data.
func UpdaterSet(_ context.Context, vulns []*claircore.Vulnerability) (driver.UpdaterSet, error) {
	res := driver.NewUpdaterSet()
	err := res.Add(&updater{data: vulns})
	if err != nil {
		return res, err
	}
	return res, nil
}

type updater struct {
	data []*claircore.Vulnerability
}

// Name provides a name for the updater.
func (u *updater) Name() string { return `ManualUpdater` }

// Fetch returns nil values as the manual updater does not fetch data.
func (u *updater) Fetch(_ context.Context, _ driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	return nil, "", nil
}

// Parse returns the provided vulnerability data or defaults to manuallyEnrichedVulns.
func (u *updater) Parse(_ context.Context, _ io.ReadCloser) ([]*claircore.Vulnerability, error) {
	if u.data == nil || len(u.data) == 0 {
		return manuallyEnrichedVulns, nil
	}
	return u.data, nil
}
