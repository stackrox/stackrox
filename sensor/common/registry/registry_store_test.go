package registry

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fakeImgName = &storage.ImageName{
		Registry: "quay.io",
		Remote:   "rhacs-eng/sandbox",
		Tag:      "noexist:1.2.0",
		FullName: "quay.io/rhacs-eng/sandbox/noexist:1.20",
	}
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

// TestRegistryStore_SpecificNamespace tests interactions with the registry store
// using an explicitly provided namespace (vs. inferred)
func TestRegistryStore_SpecificNamespace(t *testing.T) {
	ctx := context.Background()
	regStore := NewRegistryStore(alwaysInsecureCheckTLS)
	dce := config.DockerConfigEntry{Username: "username", Password: "password"}
	fakeNamespace := "fake-namespace"

	require.NoError(t, regStore.UpsertRegistry(ctx, fakeNamespace, fakeImgName.GetRegistry(), dce))
	assert.True(t, regStore.HasRegistryForImageInNamespace(fakeImgName, fakeNamespace))
	reg, err := regStore.GetRegistryForImageInNamespace(fakeImgName, fakeNamespace)
	require.NoError(t, err)
	assert.Equal(t, fakeImgName.GetRegistry(), reg.Name())
	assert.Equal(t, reg.Config().Username, "username")

	// no registry should exist based on img.Remote
	assert.False(t, regStore.HasRegistryForImage(fakeImgName))
	_, err = regStore.GetRegistryForImage(fakeImgName)
	assert.Error(t, err)
}

// TestRegistryStore_MultipleSecretsSameRegistry tests that upsert overwrites
// registry entries with matching endpoints
func TestRegistryStore_MultipleSecretsSameRegistry(t *testing.T) {
	ctx := context.Background()
	regStore := NewRegistryStore(alwaysInsecureCheckTLS)
	dceA := config.DockerConfigEntry{Username: "usernameA", Password: "passwordA"}
	dceB := config.DockerConfigEntry{Username: "usernameB", Password: "passwordB"}
	fakeNamespace := "fake-namespace"

	require.NoError(t, regStore.UpsertRegistry(ctx, fakeNamespace, fakeImgName.GetRegistry(), dceA))
	reg, err := regStore.GetRegistryForImageInNamespace(fakeImgName, fakeNamespace)
	require.NoError(t, err)
	assert.Equal(t, fakeImgName.GetRegistry(), reg.Name())
	assert.Equal(t, reg.Config().Username, dceA.Username)
	assert.Equal(t, reg.Config().Password, dceA.Password)

	require.NoError(t, regStore.UpsertRegistry(ctx, fakeNamespace, fakeImgName.GetRegistry(), dceB))
	reg, err = regStore.GetRegistryForImageInNamespace(fakeImgName, fakeNamespace)
	require.NoError(t, err)
	assert.Equal(t, fakeImgName.GetRegistry(), reg.Name())
	assert.Equal(t, reg.Config().Username, dceB.Username)
	assert.Equal(t, reg.Config().Password, dceB.Password)
}

func TestRegistryStore_UpsertNoAuthRegistry(t *testing.T) {
	ctx := context.Background()
	regStore := NewRegistryStore(alwaysInsecureCheckTLS)

	reg, err := regStore.UpsertNoAuthRegistry(ctx, "fake-namespace", fakeImgName)
	require.NoError(t, err)
	assert.Equal(t, reg.Name(), fakeImgName.GetRegistry())
	assert.Empty(t, reg.Config().Username)
	assert.Empty(t, reg.Config().Password)
}

func TestRegistryStore_GetFirstRegistryForImage(t *testing.T) {
	ctx := context.Background()
	regStore := NewRegistryStore(alwaysInsecureCheckTLS)

	dce2 := config.DockerConfigEntry{Username: "username2", Password: "password2"}
	dce3 := config.DockerConfigEntry{Username: "username3", Password: "password3"}
	dce1 := config.DockerConfigEntry{Username: "username1", Password: "password1"}

	fakeNamespace1 := "fake-namespace1"
	fakeNamespace2 := "fake-namespace2"
	fakeNamespace3 := "fake-namespace3"

	require.NoError(t, regStore.UpsertRegistry(ctx, fakeNamespace1, fakeImgName.GetRegistry(), dce1))
	require.NoError(t, regStore.UpsertRegistry(ctx, fakeNamespace2, fakeImgName.GetRegistry(), dce2))
	require.NoError(t, regStore.UpsertRegistry(ctx, fakeNamespace3, fakeImgName.GetRegistry(), dce3))

	reg, err := regStore.GetFirstRegistryForImage(fakeImgName)
	require.NoError(t, err)

	assert.Equal(t, reg.Config().Username, dce1.Username)
}
