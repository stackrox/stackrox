// Package manual provides a custom updater for vulnerability scanner.
// This updater allows manual input of vulnerability data.
package manual

import (
	"context"
	"io"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
)

// Name provides the name for the updater.
const Name = `stackrox-manual`

// UpdaterSet creates a new updater set with the provided vulnerability data.
func UpdaterSet(_ context.Context) (driver.UpdaterSet, error) {
	res := driver.NewUpdaterSet()
	err := res.Add(&updater{})
	if err != nil {
		return res, err
	}
	return res, nil
}

type updater struct{}

// Name provides the name for the updater.
func (u *updater) Name() string { return Name }

// Fetch returns nil values, as the updater does not fetch data.
func (u *updater) Fetch(_ context.Context, _ driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	return nil, "", nil
}

// Parse returns the manually added vulnerabilities.
func (u *updater) Parse(_ context.Context, _ io.ReadCloser) ([]*claircore.Vulnerability, error) {
	return u.vulns(), nil
}
