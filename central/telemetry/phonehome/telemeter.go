package phonehome

import (
	mPkg "github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
)

var (
	telemeter mPkg.Telemeter
)

// Enabled returns true if telemetry data collection is enabled.
func Enabled() bool {
	return segment.Enabled()
}

// TelemeterSingleton returns the instance of the telemeter.
func TelemeterSingleton() mPkg.Telemeter {
	once.Do(func() {
		telemeter = segment.NewTelemeter(mPkg.InstanceConfig())
	})
	return telemeter
}
