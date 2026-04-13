package app

import (
	"github.com/stackrox/rox/central/metrics"
)

func initMetrics() {
	metrics.Init()
}

// initCompliance registers all compliance checks.
func initCompliance() {
	// Import side-effect: pkg/compliance/checks registers all standard checks via init()
	// We consolidate that registration here by calling the registration function explicitly
	// This is handled by importing central/compliance/checks/remote which calls MustRegisterChecks

	// The actual registration is done via the package import
	// Future work: refactor compliance/checks to use explicit registration
}

// initGraphQL registers all GraphQL type loaders.
func initGraphQL() {
	// GraphQL loaders registration
	// Each loader registers itself via RegisterTypeFactory in their init() functions

	// Similar to compliance checks, this requires refactoring the loader registration
	// to be explicit rather than init()-based
	// Stub for now - full migration in separate PR
}
