package updater

import "github.com/stackrox/rox/sensor/common"

// Component is a sensor component with support for forcing an update (instead of just at an interval)
type Component interface {
	common.SensorComponent
	ForceUpdate()
}
