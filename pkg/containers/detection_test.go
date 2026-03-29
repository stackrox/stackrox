package containers

import (
	"testing"
)

// Verify container detection runs without error. The result depends on the
// environment (container vs host runner) and both are valid CI configurations.
func TestContainerDetection(t *testing.T) {
	t.Logf("IsRunningInContainer() = %v", IsRunningInContainer())
}
