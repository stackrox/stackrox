package app

import (
	"github.com/stackrox/rox/sensor/admission-control/manager"
)

func initMetrics() {
	manager.Init()
}
