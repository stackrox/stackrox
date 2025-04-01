package containers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Assert that container detection is running correctly by checking if it returns true in CI. Expected to return
// false when run locally.
func TestContainerDetection(t *testing.T) {
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		assert.True(t, IsRunningInContainer())
	} else {
		assert.False(t, IsRunningInContainer())
	}
}
