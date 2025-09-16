package google

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleMatch(t *testing.T) {
	// Registry integrated is for us.gcr.io and acs-san-stackroxci
	cases := []struct {
		name          *storage.ImageName
		projectMatch  bool
		registryMatch bool
	}{
		{
			name: &storage.ImageName{
				Registry: "",
				Remote:   "",
			},
			projectMatch:  false,
			registryMatch: false,
		},
		{
			name: &storage.ImageName{
				Registry: "gcr.io",
				Remote:   "acs-san-stackroxci/nginx",
			},
			projectMatch:  false, // does not match gcr.io, matches us.gcr.io
			registryMatch: false, // does not match gcr.io, matches us.gcr.io
		},
		{
			name: &storage.ImageName{
				Registry: "us.gcr.io",
				Remote:   "acs-san-stackroxci/nginx",
			},
			projectMatch:  true, // matches both us.gcr.io and acs-san-stackroxci
			registryMatch: true, // matches us.gcr.io
		},
		{
			name: &storage.ImageName{
				Registry: "us.gcr.io",
				Remote:   "acs-san-stackroxci/nginx/another",
			},
			projectMatch:  true, // matches both us.gcr.io and acs-san-stackroxci
			registryMatch: true, // matches us.gcr.io
		},
		{
			name: &storage.ImageName{
				Registry: "us.gcr.io",
				Remote:   "stackrox-ci/nginx/another",
			},
			projectMatch:  false, // matches us.gcr.io, but not stackrox-ci
			registryMatch: true,  // matches us.gcr.io
		},
	}
	reg, err := docker.NewDockerRegistryWithConfig(
		&docker.Config{Endpoint: "us.gcr.io"},
		&storage.ImageIntegration{},
		nil,
	)
	require.NoError(t, err)

	gr := &googleRegistry{
		Registry: reg,
		project:  "acs-san-stackroxci",
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s/%s - project scope", c.name.GetRegistry(), c.name.GetRemote()), func(t *testing.T) {
			assert.Equal(t, c.projectMatch, gr.Match(c.name))
		})
	}

	// Should match all projects in the registry.
	grGlobal := &googleRegistry{
		Registry: reg,
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s/%s - global scope", c.name.GetRegistry(), c.name.GetRemote()), func(t *testing.T) {
			assert.Equal(t, c.registryMatch, grGlobal.Match(c.name))
		})
	}
}

func TestGoogleValidate(t *testing.T) {
	cases := []struct {
		name    string
		config  *storage.GoogleConfig
		isValid bool
	}{
		{
			name:    "static credentials - success",
			config:  &storage.GoogleConfig{Endpoint: "eu.gcr.io", ServiceAccount: `{"type": "service_account"}`},
			isValid: true,
		},
		{
			name:    "static credentials - no endpoint",
			config:  &storage.GoogleConfig{Endpoint: "", ServiceAccount: `{"type": "service_account"}`},
			isValid: false,
		},
		{
			name:    "static credentials - no service account",
			config:  &storage.GoogleConfig{Endpoint: "eu.gcr.io", ServiceAccount: ""},
			isValid: false,
		},
		{
			name:    "workload identity - success",
			config:  &storage.GoogleConfig{Endpoint: "eu.gcr.io", ServiceAccount: "", WifEnabled: true},
			isValid: true,
		},
		{
			name:    "workload identity - no endpoint",
			config:  &storage.GoogleConfig{Endpoint: "", ServiceAccount: "", WifEnabled: true},
			isValid: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validate(c.config)
			if c.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
