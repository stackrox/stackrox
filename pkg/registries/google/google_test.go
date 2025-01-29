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
		name    *storage.ImageName
		matches bool
	}{
		{
			name: &storage.ImageName{
				Registry: "",
				Remote:   "",
			},
			matches: false,
		},
		{
			name: &storage.ImageName{
				Registry: "gcr.io",
				Remote:   "acs-san-stackroxci/nginx",
			},
			matches: false, // does not match gcr.io, matches us.gcr.io
		},
		{
			name: &storage.ImageName{
				Registry: "us.gcr.io",
				Remote:   "acs-san-stackroxci/nginx",
			},
			matches: true, // matches both us.gcr.io and acs-san-stackroxci
		},
		{
			name: &storage.ImageName{
				Registry: "us.gcr.io",
				Remote:   "acs-san-stackroxci/nginx/another",
			},
			matches: true, // matches both us.gcr.io and acs-san-stackroxci
		},
		{
			name: &storage.ImageName{
				Registry: "us.gcr.io",
				Remote:   "stackrox-ci/nginx/another",
			},
			matches: false, // matches us.gcr.io, but not stackrox-ci
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
		t.Run(fmt.Sprintf("%s/%s", c.name.GetRegistry(), c.name.GetRemote()), func(t *testing.T) {
			assert.Equal(t, c.matches, gr.Match(c.name))
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
