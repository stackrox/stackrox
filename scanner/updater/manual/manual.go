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
var Factory driver.UpdaterSetFactory = &factory{}

type factory struct{}

func (factory) UpdaterSet(context.Context) (s driver.UpdaterSet, err error) {
	s = driver.NewUpdaterSet()
	s.Add(&updater{})
	return s, nil
}

type updater struct {
}

var _ driver.Updater = (*updater)(nil)

func (u *updater) Name() string { return `manual updater` }

// Configure implements driver.Configurable.
func (u *updater) Configure(ctx context.Context, f driver.ConfigUnmarshaler, c *http.Client) error {
	ctx = zlog.ContextWithValues(ctx, "component", "updater/manual/updater.Configure")

	//client is always nil since there's no need to have http connection

	zlog.Debug(ctx).Msg("loaded incoming config")
	return nil
}

func (u *updater) Fetch(ctx context.Context, f driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	return nil, "", nil
}

func (u *updater) Parse(ctx context.Context, contents io.ReadCloser) ([]*claircore.Vulnerability, error) {
	return manuallyEnrichedVulns, nil
}
