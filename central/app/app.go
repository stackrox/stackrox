package app

import (
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/compliance/checks"
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

	metrics.Init()
	loaders.Init()
	checks.Init()
}
