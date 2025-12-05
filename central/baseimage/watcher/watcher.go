package watcher

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
)

// Enabled returns true if the base image watcher should be started.
// The watcher is enabled when:
//   - ROX_BASE_IMAGE_DETECTION feature flag is true, AND
//   - ROX_BASE_IMAGE_WATCHER_ENABLED is true (defaults to true)
func Enabled() bool {
	return features.BaseImageDetection.Enabled() && env.BaseImageWatcherEnabled.BooleanSetting()
}

// Watcher runs periodic polling of base image repositories to discover new tags.
// It follows the standard StackRox background worker pattern:
// - Start() spawns goroutines and returns immediately
// - Stop() signals shutdown and blocks until cleanup completes
//
// The watcher integrates with Central's lifecycle via singleton pattern.
// It is started in central/main.go after database initialization and stopped
// during Central shutdown.
//
//go:generate mockgen-wrapper
type Watcher interface {
	// Start spawns background goroutines for periodic polling.
	// Returns immediately without blocking.
	// Safe to call multiple times (subsequent calls are no-ops).
	Start()

	// Stop signals the watcher to shut down and blocks until all
	// goroutines have exited cleanly.
	// Safe to call multiple times (subsequent calls are no-ops).
	Stop()
}
