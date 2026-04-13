package app

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/premain"
)

var (
	log = logging.LoggerForModule()
)

// Run is the main entry point for the central application.
// For Phase 1, this just does early initialization.
// The actual central logic remains in main.CentralRun() until full migration.
func Run() {
	memlimit.SetMemoryLimit()
	premain.StartMain()

	// Component-specific initialization will be added in Phase 3
	initMetrics()
	initCompliance()
	initGraphQL()
}
