package bundle

import (
	"github.com/stackrox/stackrox/sensor/upgrader/upgradectx"
)

// FetchBundle fetches the sensor bundle from central, and returns a view of its contents.
func FetchBundle(ctx *upgradectx.UpgradeContext) (Contents, error) {
	f := &fetcher{ctx: ctx}
	return f.FetchBundle()
}
