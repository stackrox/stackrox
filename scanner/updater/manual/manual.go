package manual

import (
	"context"
	"io"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
)

// Factory is the UpdaterSetFactory exposed by this package.
//
// All configuration is done on the returned updaters.
type Factory struct {
}

func UpdaterSet(_ context.Context, vulns []*claircore.Vulnerability) (driver.UpdaterSet, error) {
	res := driver.NewUpdaterSet()
	res.Add(&updater{data: vulns})
	return res, nil
}

var _ driver.Updater = (*updater)(nil)

type updater struct {
	data []*claircore.Vulnerability
}

func (u *updater) Name() string { return `manual updater` }

func (u *updater) Fetch(_ context.Context, _ driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	return nil, "", nil
}

func (u *updater) Parse(_ context.Context, _ io.ReadCloser) ([]*claircore.Vulnerability, error) {
	if u.data == nil || len(u.data) == 0 {
		return manuallyEnrichedVulns, nil
	}
	return u.data, nil
}
