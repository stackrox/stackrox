package registry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/rhel"
	"github.com/stackrox/rox/pkg/sync"
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
	reg, err := regStore.GetRegistryForImageInNamespace(img, "qa")
	require.NoError(t, err)
	assert.Equal(t, "image-registry.openshift-image-registry.svc:5000", reg.Config().RegistryHostname)

	img = &storage.ImageName{
		Registry: "image-registry.openshift-image-registry.svc.local:5000",
		Remote:   "qa/nginx",
		Tag:      "nginx:1.18.0",
		FullName: "image-registry.openshift-image-registry.svc.local:5000/qa/nginx:1.18.0",
	}
	reg, err = regStore.GetRegistryForImageInNamespace(img, "qa")
	require.NoError(t, err)
	assert.Equal(t, "image-registry.openshift-image-registry.svc.local:5000", reg.Config().RegistryHostname)

	img = &storage.ImageName{
		Registry: "172.99.12.11:5000",
		Remote:   "qa/nginx",
		Tag:      "nginx:1.18.0",
		FullName: "172.99.12.11:5000/qa/nginx:1.18.0",
	}
	reg, err = regStore.GetRegistryForImageInNamespace(img, "qa")
	require.NoError(t, err)
	assert.Equal(t, "172.99.12.11:5000", reg.Config().RegistryHostname)
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
	assert.Equal(t, fakeImgName.GetRegistry(), reg.Config().RegistryHostname)
	assert.Equal(t, reg.Config().Username, "username")

	// no registry should exist based on img.Remote
	_, err = regStore.GetRegistryForImageInNamespace(fakeImgName, "qa")
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
	assert.Equal(t, fakeImgName.GetRegistry(), reg.Config().RegistryHostname)
	assert.Equal(t, reg.Config().Username, dceA.Username)
	assert.Equal(t, reg.Config().Password, dceA.Password)

	require.NoError(t, regStore.UpsertRegistry(ctx, fakeNamespace, fakeImgName.GetRegistry(), dceB))
	reg, err = regStore.GetRegistryForImageInNamespace(fakeImgName, fakeNamespace)
	require.NoError(t, err)
	assert.Equal(t, fakeImgName.GetRegistry(), reg.Config().RegistryHostname)
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
	ii := createImageIntegration("http://example.com", config.DockerConfigEntry{}, false, "")
	assert.Equal(t, ii.Type, docker.GenericDockerRegistryType)

	ii = createImageIntegration("https://registry.redhat.io", config.DockerConfigEntry{}, true, "")
	assert.Equal(t, ii.Type, rhel.RedHatRegistryType)
}

func TestRegistryStore_IsLocal(t *testing.T) {
	regStore := NewRegistryStore(alwaysInsecureCheckTLS)
	regStore.AddClusterLocalRegistryHost("image-registry.openshift-image-registry.svc:5000")

	specificRegs := []*central.DelegatedRegistryConfig_DelegatedRegistry{
		{Path: "isfound.svc/repo/path"},
		{Path: "otherfound.svc"},
	}

	tt := map[string]struct {
		image    *storage.ImageName
		config   *central.DelegatedRegistryConfig
		expected bool
	}{
		"nil": {
			image:    nil,
			config:   nil,
			expected: false,
		},
		"cluster local": {
			image: &storage.ImageName{
				Registry: "image-registry.openshift-image-registry.svc:5000",
			},
			config:   nil,
			expected: true,
		},
		"nil config": {
			image: &storage.ImageName{
				Registry: "noexist.svc",
			},
			config:   nil,
			expected: false,
		},
		"enabled for none": {
			image: &storage.ImageName{
				Registry: "noexist.svc",
			},
			config: &central.DelegatedRegistryConfig{
				EnabledFor: central.DelegatedRegistryConfig_NONE,
			},
			expected: false,
		},
		"enabled for all": {
			image: &storage.ImageName{
				Registry: "noexist.svc",
			},
			config: &central.DelegatedRegistryConfig{
				EnabledFor: central.DelegatedRegistryConfig_ALL,
			},
			expected: true,
		},
		"specific not found": {
			image: &storage.ImageName{
				Registry: "isnotfound.svc",
				FullName: "isnotfound.svc/repo/path",
			},
			config: &central.DelegatedRegistryConfig{
				EnabledFor: central.DelegatedRegistryConfig_SPECIFIC,
				Registries: specificRegs,
			},
			expected: false,
		},
		"specific found by host": {
			image: &storage.ImageName{
				Registry: "otherfound.svc",
				FullName: "otherfound.svc/random/path",
			},
			config: &central.DelegatedRegistryConfig{
				EnabledFor: central.DelegatedRegistryConfig_SPECIFIC,
				Registries: specificRegs,
			},
			expected: true,
		},
		"specific found by path": {
			image: &storage.ImageName{
				Registry: "isfound.svc",
				FullName: "isfound.svc/repo/path",
			},
			config: &central.DelegatedRegistryConfig{
				EnabledFor: central.DelegatedRegistryConfig_SPECIFIC,
				Registries: specificRegs,
			},
			expected: true,
		},
		"specific not found by path": {
			image: &storage.ImageName{
				Registry: "isfound.svc",
				FullName: "isfound.svc/notfound/repo/path",
			},
			config: &central.DelegatedRegistryConfig{
				EnabledFor: central.DelegatedRegistryConfig_SPECIFIC,
				Registries: specificRegs,
			},
			expected: false,
		},
	}

	for name, test := range tt {
		tf := func(t *testing.T) {
			regStore.SetDelegatedRegistryConfig(test.config)
			r := regStore.IsLocal(test.image)

			assert.Equal(t, test.expected, r)
		}

		t.Run(name, tf)
	}

}

func TestRegistryStore_GenImgIntName(t *testing.T) {
	tt := []struct {
		prefix    string
		namespace string
		registry  string
		expected  string
	}{
		{"", "", "", ""},
		{"PRE", "", "", "PRE"},
		{"PRE", "", "REG", "PRE/reg:REG"},
		{"PRE", "NAME", "", "PRE/ns:NAME"},
		{"PRE", "NAME", "REG", "PRE/ns:NAME/reg:REG"},
	}

	for i, test := range tt {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			actual := genIntegrationName(test.prefix, test.namespace, test.registry)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestDataRaceAtCleanup(_ *testing.T) {
	testNamespace := "test-ns"
	regStore := NewRegistryStore(alwaysInsecureCheckTLS)
	regStore.store[testNamespace] = registries.NewSet(regStore.factory)
	wg := sync.WaitGroup{}
	doneSignal := concurrency.NewSignal()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-doneSignal.Done():
				return
			default:
				// random reads
				regStore.getRegistries(testNamespace)
				regStore.IsLocal(&storage.ImageName{})
				regStore.GetMatchingCentralRegistryIntegrations(&storage.ImageName{})
				_, _ = regStore.GetGlobalRegistryForImage(&storage.ImageName{})
			}
		}
	}()
	time.Sleep(10 * time.Millisecond)
	regStore.Cleanup()
	doneSignal.Signal()
	wg.Wait()
}

func TestReconcile(t *testing.T) {
	regStore := NewRegistryStore(alwaysInsecureCheckTLS)
	testCases := map[string]bool{
		"a": false,
		"b": true,
		"c": true,
	}

	for secretID, isRemoved := range testCases {
		regStore.AddSecretID(secretID)
		if isRemoved {
			regStore.RemoveSecretID(secretID)
		}
	}

	for secretID, isRemoved := range testCases {
		id, err := regStore.ReconcileDelete("", secretID, 0)
		require.NoError(t, err)
		if isRemoved {
			assert.NotEmpty(t, id)
		} else {
			assert.Empty(t, id)
		}
	}

}
