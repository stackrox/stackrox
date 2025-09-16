package google

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGoogleValidate(t *testing.T) {
	cases := []struct {
		name    string
		config  *storage.GoogleConfig
		isValid bool
	}{
		{
			name: "static credentials - success",
			config: &storage.GoogleConfig{
				Endpoint:       "eu.gcr.io",
				Project:        "test-project",
				ServiceAccount: `{"type": "service_account"}`,
			},
			isValid: true,
		},
		{
			name: "static credentials - no endpoint",
			config: &storage.GoogleConfig{
				Endpoint:       "",
				Project:        "test-project",
				ServiceAccount: `{"type": "service_account"}`,
			},
			isValid: false,
		},
		{
			name: "static credentials - no project",
			config: &storage.GoogleConfig{
				Endpoint:       "eu.gcr.io",
				Project:        "",
				ServiceAccount: `{"type": "service_account"}`,
			},
			isValid: false,
		},
		{
			name: "static credentials - no service account",
			config: &storage.GoogleConfig{
				Endpoint:       "eu.gcr.io",
				Project:        "test-project",
				ServiceAccount: "",
			},
			isValid: false,
		},
		{
			name: "workload identity - rejected",
			config: &storage.GoogleConfig{
				Endpoint:       "eu.gcr.io",
				Project:        "test-project",
				ServiceAccount: `{"type": "service_account"}`,
				WifEnabled:     true,
			},
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
