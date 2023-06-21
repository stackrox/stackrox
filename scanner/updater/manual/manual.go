package manual

import (
	"context"
	"io"
	"net/http"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
)

// Factory is the UpdaterSetFactory exposed by this package.
//
// All configuration is done on the returned updaters. See the [Config] type.
type Factory struct {
}

func UpdaterSet(ctx context.Context, vulns []*claircore.Vulnerability) (driver.UpdaterSet, error) {
	res := driver.NewUpdaterSet()
	if vulns != nil && len(vulns) > 0 {
		res.Add(&updater{data: vulns})
	} else {
		res.Add(&updater{})
	}
	return res, nil
}

type updater struct {
	data []*claircore.Vulnerability
}

var _ driver.Updater = (*updater)(nil)

func (u *updater) Name() string { return `manual updater` }

// Configure implements driver.Configurable.
func (u *updater) Configure(ctx context.Context, f driver.ConfigUnmarshaler, c *http.Client) error {
	ctx = zlog.ContextWithValues(ctx, "component", "updater/manual/updater.Configure")

	zlog.Debug(ctx).Msg("loaded incoming config")
	return nil
}

func (u *updater) Fetch(_ context.Context, _ driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	return nil, "", nil
}

func (u *updater) Parse(_ context.Context, _ io.ReadCloser) ([]*claircore.Vulnerability, error) {
	if u.data == nil || len(u.data) == 0 {
		return manuallyEnrichedVulns, nil
	}
	return u.data, nil
}
