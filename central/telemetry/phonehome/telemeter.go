package phonehome

import (
	"github.com/stackrox/rox/pkg/sync"
	pkgPH "github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/segment"
)

var (
	telemeter     pkgPH.Telemeter
	onceTelemeter sync.Once
)

// Enabled returns true if telemetry data collection is enabled.
func Enabled() bool {
	return segment.Enabled()
}

// TelemeterSingleton returns the instance of the telemeter.
func TelemeterSingleton() pkgPH.Telemeter {
	onceTelemeter.Do(func() {
		cfg := pkgPH.InstanceConfig()
		telemeter = segment.NewTelemeter(cfg.CentralID, cfg.Identity)
	})
	return telemeter
}
