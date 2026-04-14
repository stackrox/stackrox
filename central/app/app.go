package app

import (
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/premain"
)

// Run is the main entry point for the central application.
// Performs early initialization and component-specific setup before
// main.centralRun() starts the actual central service logic.
func Run() {
	memlimit.SetMemoryLimit()
	premain.StartMain()

	loaders.Init()
}
