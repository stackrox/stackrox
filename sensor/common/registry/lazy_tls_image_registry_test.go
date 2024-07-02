package registry

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	imgTypes "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
	regMocks "github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNoPanics(t *testing.T) {
	assert.NotPanics(t, func() {
		failCache := newTLSCheckCache(alwaysFailCheckTLS)

		lazyRegistry := &lazyTLSCheckRegistry{tlsCheckCache: failCache}
		lazyRegistry.Config()
		lazyRegistry.DataSource()
		lazyRegistry.Match(nil)
		_, _ = lazyRegistry.Metadata(nil)
		lazyRegistry.Name()
		lazyRegistry.Source()
	})
}

func TestConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockReg := regMocks.NewMockRegistry(ctrl)
	creator := func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
		cfg := integration.GetIntegrationConfig().(*storage.ImageIntegration_Docker).Docker
		mockReg.EXPECT().Config().Return(&types.Config{
			Insecure: cfg.Insecure,
		})

		return mockReg, nil
	}

	source := genImageIntegration("example.com")
	reg, err := createReg(source, creator, alwaysInsecureCheckTLS)
	require.NoError(t, err)

	// Before init Insecure should be at the default value (false).
	assert.False(t, reg.Config().GetInsecure())

	// Simulate some other method triggering lazy init.
	reg.lazyInit()

	// After init Insecure should come from the backing registry.
	assert.True(t, reg.Config().GetInsecure())
}

func TestDataSource(t *testing.T) {
	ds := &storage.DataSource{}
	reg := &lazyTLSCheckRegistry{
		dataSource: ds,
	}

	got := reg.DataSource()
	if got != ds {
		t.Error(t, "Recieved different datasource than expected.")
	}
}

func TestHTTPClient(t *testing.T) {
	reg := &lazyTLSCheckRegistry{}
	if !buildinfo.ReleaseBuild {
		require.PanicsWithError(t, "not implemented", func() { reg.HTTPClient() })
	} else {
		assert.NotPanics(t, func() { reg.HTTPClient() })
	}
}

func TestMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockReg := regMocks.NewMockRegistry(ctrl)
	creator := func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
		return mockReg, nil
	}

	source := genImageIntegration("example.com")
	reg, err := createReg(source, creator, alwaysInsecureCheckTLS)
	require.NoError(t, err)

	imgName, _, err := utils.GenerateImageNameFromString("example.com/repo/path:latest")
	require.NoError(t, err)
	assert.True(t, reg.Match(imgName))
	assert.False(t, reg.initialized)

	imgName, _, err = utils.GenerateImageNameFromString("example.net/repo/path:latest")
	require.NoError(t, err)
	assert.False(t, reg.Match(imgName))
	assert.False(t, reg.initialized)
}

func TestMetadata(t *testing.T) {
	cImg, err := utils.GenerateImageFromString("example.com/repo/path:latest")
	require.NoError(t, err)
	img := imgTypes.ToImage(cImg)

	t.Run("error when TLS check fails", func(t *testing.T) {
		source := genImageIntegration("example.com")

		reg, err := createReg(source, nil, alwaysFailCheckTLS)
		require.NoError(t, err)

		m, err := reg.Metadata(img)
		require.ErrorContains(t, err, "fake")
		assert.Nil(t, m)
		// Confirm that initialization is NOT considered complete due to TLS checks
		// being temporal.
		assert.False(t, reg.initialized)

		// Subsequent call should skip the TLS check.
		m, err = reg.Metadata(img)
		require.ErrorContains(t, err, "skipped")
		assert.Nil(t, m)
	})

	t.Run("error when creator fails", func(t *testing.T) {
		creator := func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			return nil, errors.New("fake creator error")
		}
		source := genImageIntegration("example.com")

		reg, err := createReg(source, creator, alwaysInsecureCheckTLS)
		require.NoError(t, err)

		m, err := reg.Metadata(img)
		require.ErrorContains(t, err, "fake creator error")
		assert.Nil(t, m)

		// Confirm that initialization IS considered complete.
		assert.True(t, reg.initialized)

		// Repeat to make sure same error is returned.
		m, err = reg.Metadata(img)
		require.ErrorContains(t, err, "fake creator error")
		assert.Nil(t, m)
	})

	t.Run("successful lazy init", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockReg := regMocks.NewMockRegistry(ctrl)
		fakeMetadata := &storage.ImageMetadata{}
		creator := func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			return mockReg, nil
		}

		source := genImageIntegration("example.com")

		reg, err := createReg(source, creator, alwaysInsecureCheckTLS)
		require.NoError(t, err)

		mockReg.EXPECT().Metadata(img).Return(fakeMetadata, nil)
		m, err := reg.Metadata(img)
		require.NoError(t, err)
		assert.True(t, reg.initialized)
		protoassert.Equal(t, fakeMetadata, m)

		mockReg.EXPECT().Metadata(img).Return(fakeMetadata, nil)
		m, err = reg.Metadata(img)
		require.NoError(t, err)
		protoassert.Equal(t, fakeMetadata, m)
	})
}

func TestName(t *testing.T) {
	ii := genImageIntegration("example.com")
	reg, err := createReg(ii, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "fake-name", reg.Name())
}

func TestSource(t *testing.T) {
	ii := genImageIntegration("example.com")
	reg, err := createReg(ii, nil, nil)
	require.NoError(t, err)

	got := reg.Source()
	if got != ii {
		t.Error(t, "Recieved different datasource than expected.")
	}
}

func TestTest(t *testing.T) {
	reg := &lazyTLSCheckRegistry{}
	if !buildinfo.ReleaseBuild {
		require.PanicsWithError(t, "not implemented", func() { _ = reg.Test() })
	} else {
		assert.NotPanics(t, func() { _ = reg.Test() })
	}
}

func genImageIntegration(endpoint string) *storage.ImageIntegration {
	return &storage.ImageIntegration{
		Id:   "fake-id",
		Name: "fake-name",
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: endpoint,
			},
		},
	}
}

func createReg(source *storage.ImageIntegration, creator types.Creator, tlsCheckFunc CheckTLS) (*lazyTLSCheckRegistry, error) {
	cfg, err := extractDockerConfig(source)
	if err != nil {
		return nil, err
	}

	url := docker.FormatURL(cfg.GetEndpoint())
	host := docker.RegistryServer(cfg.GetEndpoint(), url)

	reg := &lazyTLSCheckRegistry{
		source:           source,
		creator:          creator,
		dockerConfig:     cfg,
		url:              url,
		registryHostname: host,
		dataSource: &storage.DataSource{
			Id:   source.GetId(),
			Name: source.GetName(),
		},
		tlsCheckCache: newTLSCheckCache(tlsCheckFunc),
	}

	return reg, nil
}
