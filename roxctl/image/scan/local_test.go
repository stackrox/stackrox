package scan

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocal(t *testing.T) {
	cmd := imageScanCommand{
		image:      "test:0.0.1",
		retryDelay: 3,
		retryCount: 3,
		timeout:    1 * time.Minute,
	}
	result, err := cmd.scanLocal()
	require.NoError(t, err)
	assert.Equal(t, "busybox@sha256:b5d6fe0712636ceb7430189de28819e195e8966372edfc2d9409d79402a0dc16", result.Name.FullName)
}

func TestImage(t *testing.T) {
	img, err := NewImage("test:0.0.1")
	require.NoError(t, err)
	manifest, err := img.GetManifest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "sha256:7cfbbec8963d8f13e6c70416d6592e1cc10f47a348131290a55d43c3acab3fb9", manifest.Hash.String())
	assert.Len(t, manifest.Layers, 1)
}

func TestImage1(t *testing.T) {
	img, err := NewImage("busybox@sha256:b5d6fe0712636ceb7430189de28819e195e8966372edfc2d9409d79402a0dc16")
	require.NoError(t, err)
	manifest, err := img.GetManifest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "sha256:7cfbbec8963d8f13e6c70416d6592e1cc10f47a348131290a55d43c3acab3fb9", manifest.Hash.String())
	assert.Len(t, manifest.Layers, 1)
}
