package containers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainerDetection(t *testing.T) {
	if !IsRunningInContainer() {
		t.Skip("not running in a container")
	}
	assert.True(t, IsRunningInContainer())
}
