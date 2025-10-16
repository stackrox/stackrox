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
			config: storage.GoogleConfig_builder{
				Endpoint:       "eu.gcr.io",
				Project:        "test-project",
				ServiceAccount: `{"type": "service_account"}`,
			}.Build(),
			isValid: true,
		},
		{
			name: "static credentials - no endpoint",
			config: storage.GoogleConfig_builder{
				Endpoint:       "",
				Project:        "test-project",
				ServiceAccount: `{"type": "service_account"}`,
			}.Build(),
			isValid: false,
		},
		{
			name: "static credentials - no project",
			config: storage.GoogleConfig_builder{
				Endpoint:       "eu.gcr.io",
				Project:        "",
				ServiceAccount: `{"type": "service_account"}`,
			}.Build(),
			isValid: false,
		},
		{
			name: "static credentials - no service account",
			config: storage.GoogleConfig_builder{
				Endpoint:       "eu.gcr.io",
				Project:        "test-project",
				ServiceAccount: "",
			}.Build(),
			isValid: false,
		},
		{
			name: "workload identity - rejected",
			config: storage.GoogleConfig_builder{
				Endpoint:       "eu.gcr.io",
				Project:        "test-project",
				ServiceAccount: `{"type": "service_account"}`,
				WifEnabled:     true,
			}.Build(),
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
