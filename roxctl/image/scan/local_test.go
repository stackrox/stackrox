package scan

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocal(t *testing.T) {
	cmd := imageScanCommand{
		image:      "test:0.0.1",
		retryDelay: 3,
		retryCount: 3,
		timeout:    1 * time.Minute,
	}
	result, err := cmd.scanLocal()
	assert.NoError(t, err)
	assert.Equal(t, "busybox@sha256:b5d6fe0712636ceb7430189de28819e195e8966372edfc2d9409d79402a0dc16", result.Name.FullName)
}
