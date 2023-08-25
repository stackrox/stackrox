package declarativeconfig

import "github.com/stackrox/rox/pkg/telemetry/phonehome"

// Manager manages reconciling declarative configuration.
type Manager interface {
	ReconcileDeclarativeConfigurations()
	Gather() phonehome.GatherFunc
}
