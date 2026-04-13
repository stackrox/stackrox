package app

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/profiling"
)

var (
	log = logging.LoggerForModule()
)

// Run is the main entry point for the central application.
// Performs early initialization and component-specific setup before
// main.CentralRun() starts the actual central service logic.
func Run() {
	profiling.SetComponentLabel()
	memlimit.SetMemoryLimit()
	premain.StartMain()

	initMetrics()
	initCompliance()
	initGraphQL()
}
