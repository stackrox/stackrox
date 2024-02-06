package scan

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocal(t *testing.T) {
	cmd := imageScanCommand{
		image:      "nginx@sha:256d02f9b9db4d759ef27dc26b426b842ff2fb881c5c6079612d27ec36e36b132dd",
		retryDelay: 3,
		retryCount: 3,
		timeout:    1 * time.Minute,
	}
	err := cmd.localScan()
	assert.NoError(t, err)
}
