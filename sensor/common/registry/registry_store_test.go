package registry

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// alwaysInsecureCheckTLS is an implementation of registry.CheckTLS
// which always says the given address is insecure.
func alwaysInsecureCheckTLS(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func TestRegistryStore_same_namespace(t *testing.T) {
	ctx := context.Background()

	regStore := NewRegistryStore(alwaysInsecureCheckTLS)

	dce := config.DockerConfigEntry{
		Username: "username",
		Password: "password",
	}
	require.NoError(t, regStore.UpsertRegistry(ctx, "qa", "image-registry.openshift-image-registry.svc:5000", dce))
	require.NoError(t, regStore.UpsertRegistry(ctx, "qa", "image-registry.openshift-image-registry.svc.local:5000", dce))
	require.NoError(t, regStore.UpsertRegistry(ctx, "qa", "172.99.12.11:5000", dce))

	img := &storage.ImageName{
		Registry: "image-registry.openshift-image-registry.svc:5000",
		Remote:   "qa/nginx",
		Tag:      "nginx:1.18.0",
		FullName: "image-registry.openshift-image-registry.svc:5000/qa/nginx:1.18.0",
	}
	assert.True(t, regStore.HasRegistryForImage(img))
	reg, err := regStore.GetRegistryForImage(img)
	require.NoError(t, err)
	assert.Equal(t, "image-registry.openshift-image-registry.svc:5000", reg.Name())

	img = &storage.ImageName{
		Registry: "image-registry.openshift-image-registry.svc.local:5000",
		Remote:   "qa/nginx",
		Tag:      "nginx:1.18.0",
		FullName: "image-registry.openshift-image-registry.svc.local:5000/qa/nginx:1.18.0",
	}
	assert.True(t, regStore.HasRegistryForImage(img))
	reg, err = regStore.GetRegistryForImage(img)
	require.NoError(t, err)
	assert.Equal(t, "image-registry.openshift-image-registry.svc.local:5000", reg.Name())

	img = &storage.ImageName{
		Registry: "172.99.12.11:5000",
		Remote:   "qa/nginx",
		Tag:      "nginx:1.18.0",
		FullName: "172.99.12.11:5000/qa/nginx:1.18.0",
	}
	assert.True(t, regStore.HasRegistryForImage(img))
	reg, err = regStore.GetRegistryForImage(img)
	require.NoError(t, err)
	assert.Equal(t, "172.99.12.11:5000", reg.Name())
}
