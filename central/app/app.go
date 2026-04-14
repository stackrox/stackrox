package app

import (
	centralChecks "github.com/stackrox/rox/central/compliance/checks"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	backupPlugins "github.com/stackrox/rox/central/externalbackups/plugins/all"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/pkg/compliance/checks"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/premain"
)

// Run is the main entry point for the central application.
// Performs early initialization and component-specific setup before
// main.CentralRun() starts the actual central service logic.
func Run() {
	memlimit.SetMemoryLimit()
	premain.StartMain()

	metrics.Init()
	loaders.Init()
	checks.Init()
	centralChecks.Init()
	notifiers.Init()
	metadata.Init()
	backupPlugins.Init()
}
