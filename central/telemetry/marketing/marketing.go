package marketing

import (
	mPkg "github.com/stackrox/rox/pkg/telemetry/marketing"
	"github.com/stackrox/rox/pkg/telemetry/marketing/segment"
)

var (
	telemeter mPkg.Telemeter
)

// Enabled returns true if marketing telemetry data collection is enabled.
func Enabled() bool {
	return segment.Enabled()
}

func Stop() {
	m.telemeter.Stop()
}

func TelemeterSingleton() mPkg.Telemeter {
	once.Do(func() {
		telemeter = segment.Init(mPkg.Singleton())
	})
	return telemeter
}
