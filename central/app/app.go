package app

import (
	"github.com/stackrox/rox/central/metrics"
	grpcmetrics "github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/images"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/rate"
	"github.com/stackrox/rox/pkg/tlscheckcache"
)

// Run is the main entry point for the central application.
// Performs early initialization and component-specific setup before
// main.centralRun() starts the actual central service logic.
func Run() {
	memlimit.SetMemoryLimit()
	premain.StartMain()

	metrics.Init()
	postgres.Init()
	rate.Init()
	images.Init()
	tlscheckcache.Init()
	grpcmetrics.Init()
}
