package containers

import (
	"testing"
)

// Smoke test: IsRunningInContainer should not panic regardless of environment.
// The result depends on whether CI runs in a container or on a host runner —
// both are valid configurations.
func TestContainerDetection(t *testing.T) {
	result := IsRunningInContainer()
	t.Logf("IsRunningInContainer() = %v", result)
}
