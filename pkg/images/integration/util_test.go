package integration

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRegistry struct {
	registryTypes.ImageRegistry
	match            bool
	registryHostName string
}

func (f *fakeRegistry) Match(_ *storage.ImageName) bool {
	return f.match
}

func (f *fakeRegistry) Config() *registryTypes.Config {
	if f.registryHostName == "" {
		return nil
	}
	return &registryTypes.Config{
		RegistryHostname: f.registryHostName,
	}
}

func TestGetMatchingImageIntegrations(t *testing.T) {
	imgName, _, err := imgUtils.GenerateImageNameFromString("docker.io/nginx:1.23")
	require.NoError(t, err)

	cases := map[string]struct {
		registries         []registryTypes.ImageRegistry
		expectedRegistries []registryTypes.ImageRegistry
	}{
		"no matches for image": {
			registries: []registryTypes.ImageRegistry{
				&fakeRegistry{match: false},
			},
		},
		"single matche for image name": {
			registries: []registryTypes.ImageRegistry{
				&fakeRegistry{match: true},
			},
			expectedRegistries: []registryTypes.ImageRegistry{
				&fakeRegistry{match: true},
			},
		},
		"multiple matches for image name": {
			registries: []registryTypes.ImageRegistry{
				&fakeRegistry{match: true, registryHostName: "docker.io"},
				&fakeRegistry{match: true},
				&fakeRegistry{match: true, registryHostName: "hub.docker.io"},
			},
			expectedRegistries: []registryTypes.ImageRegistry{
				&fakeRegistry{match: true},
				&fakeRegistry{match: true, registryHostName: "docker.io"},
				&fakeRegistry{match: true, registryHostName: "hub.docker.io"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			res := GetMatchingImageIntegrations(tc.registries, imgName)
			assert.Equal(t, tc.expectedRegistries, res)
		})
	}
}
