package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/rhel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fakeImgName = &storage.ImageName{
		Registry: "example.com",
		Remote:   "rhacs-eng/sandbox",
		Tag:      "noexist",
		FullName: "example.com/rhacs-eng/sandbox:noexist",
	}
)

// alwaysInsecureCheckTLS is an implementation of registry.CheckTLS
// which always says the given address is insecure.
func alwaysInsecureCheckTLS(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func alwaysFailCheckTLS(_ context.Context, _ string) (bool, error) {
	return false, errors.New("fake tls failure")
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

func TestRegistryStore_GlobalStore(t *testing.T) {
	ctx := context.Background()
	regStore := NewRegistryStore(alwaysInsecureCheckTLS)
	dce := config.DockerConfigEntry{Username: "username", Password: "password"}

	_, err := regStore.GetGlobalRegistryForImage(fakeImgName)
	require.Error(t, err, "error is expected on empty store")

	err = regStore.UpsertGlobalRegistry(ctx, fakeImgName.GetRegistry(), dce)
	require.NoError(t, err, "should be no error on valid upsert")

	reg, err := regStore.GetGlobalRegistryForImage(fakeImgName)
	require.NoError(t, err, "should be no error on valid get")
	assert.NotNil(t, reg)
	assert.Equal(t, reg.Config().Username, dce.Username)

	// sanity check
	assert.Zero(t, len(regStore.store), "non-global store should not have been modified")
}

func TestRegistryStore_GlobalStoreFailUpsertCheckTLS(t *testing.T) {
	ctx := context.Background()
	regStore := NewRegistryStore(alwaysFailCheckTLS)
	dce := config.DockerConfigEntry{Username: "username", Password: "password"}

	// upsert that fails TLS check should error out
	require.Error(t, regStore.UpsertGlobalRegistry(ctx, fakeImgName.GetRegistry(), dce))

	// sanity check
	assert.True(t, regStore.globalRegistries.IsEmpty(), "global store should not be populated")
}

func TestRegistryStore_CreateImageIntegrationType(t *testing.T) {
	ii := createImageIntegration("http://example.com", config.DockerConfigEntry{}, false)
	assert.Equal(t, ii.Type, docker.GenericDockerRegistryType)

	ii = createImageIntegration("https://registry.redhat.io", config.DockerConfigEntry{}, true)
	assert.Equal(t, ii.Type, rhel.RedHatRegistryType)
}
