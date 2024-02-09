package scan

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocal(t *testing.T) {
	// Before this test ensure you have image locally
	// docker pull  quay.io/stackrox-io/main:4.0.0
	// docker tag sha256:760780e4e49cdfd6a5a480f87a00daf30995b9fa0edc39534c86e59fb24ddc2f test:0.0.1
	images := map[string]string{
		"repo digest": "quay.io/stackrox-io/main@sha256:e7d366c7579e4e08a26c24bac03dc5f2869006c10183e3c5f780f3754f01e3c3",
		//"tag name":    "quay.io/stackrox-io/main:4.0.0", can't run multiple tsts due to store registering sql functions
		//"local tag":   "test:0.0.1",
		//"sha":         "sha256:760780e4e49cdfd6a5a480f87a00daf30995b9fa0edc39534c86e59fb24ddc2f", // not working as it gets default registry
	}
	for name, image := range images {
		t.Run(name, func(t *testing.T) {
			cmd := imageScanCommand{
				dbUri:      "testdata/vulndb",
				image:      image,
				retryDelay: 3,
				retryCount: 3,
				timeout:    1 * time.Minute,
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result, err := cmd.scanLocal(ctx)
			require.NoError(t, err)
			assert.Equal(t, image, result.Name.FullName)
			assert.Equal(t, "sha256:d407c96802e7db04ec01f267574aba3c6c0f3f445a879232f249af01e84a4f12", result.Id)
			assert.Equal(t, "linux", result.GetScan().OperatingSystem)
			assert.Len(t, result.GetScan().Components, 0)
		})
	}
}

func TestImage(t *testing.T) {
	img, err := newImage("test:0.0.1")
	require.NoError(t, err)
	manifest, err := img.GetManifest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "sha256:7cfbbec8963d8f13e6c70416d6592e1cc10f47a348131290a55d43c3acab3fb9", manifest.Hash.String())
	assert.Len(t, manifest.Layers, 1)
}

func TestImage1(t *testing.T) {
	img, err := newImage("busybox@sha256:b5d6fe0712636ceb7430189de28819e195e8966372edfc2d9409d79402a0dc16")
	require.NoError(t, err)
	manifest, err := img.GetManifest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "sha256:7cfbbec8963d8f13e6c70416d6592e1cc10f47a348131290a55d43c3acab3fb9", manifest.Hash.String())
	assert.Len(t, manifest.Layers, 1)
}
