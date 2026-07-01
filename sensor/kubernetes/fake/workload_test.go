package fake

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateWorkload(t *testing.T) {
	tests := map[string]struct {
		input       Workload
		expected    Workload
		expectError bool
	}{
		"valid workload passes unchanged": {
			input: Workload{
				OfflineModeInterval: time.Minute,
				NetworkWorkload:     NetworkWorkload{OpenPortReuseProbability: 0.5},
			},
			expected: Workload{
				OfflineModeInterval: time.Minute,
				NetworkWorkload:     NetworkWorkload{OpenPortReuseProbability: 0.5},
			},
			expectError: false,
		},
		"negative offlineModeInterval is clamped to disabled": {
			input:    Workload{OfflineModeInterval: -5 * time.Second},
			expected: Workload{OfflineModeInterval: 0},
		},
		"offlineModeInterval below minimum is clamped to minimum": {
			input:    Workload{OfflineModeInterval: time.Second},
			expected: Workload{OfflineModeInterval: 10 * time.Second},
		},
		"offlineModeInterval at minimum is left unchanged": {
			input:    Workload{OfflineModeInterval: 10 * time.Second},
			expected: Workload{OfflineModeInterval: 10 * time.Second},
		},
		"openPortReuseProbability below 0 is clamped and errors": {
			input:       Workload{NetworkWorkload: NetworkWorkload{OpenPortReuseProbability: -0.5}},
			expected:    Workload{NetworkWorkload: NetworkWorkload{OpenPortReuseProbability: 0}},
			expectError: true,
		},
		"openPortReuseProbability above 1 is clamped and errors": {
			input:       Workload{NetworkWorkload: NetworkWorkload{OpenPortReuseProbability: 1.5}},
			expected:    Workload{NetworkWorkload: NetworkWorkload{OpenPortReuseProbability: 1}},
			expectError: true,
		},
		"negative numDockerCfgSecrets is clamped and errors": {
			input:       Workload{SecretWorkload: SecretWorkload{NumDockerCfgSecrets: -3}},
			expected:    Workload{SecretWorkload: SecretWorkload{NumDockerCfgSecrets: 0}},
			expectError: true,
		},
		"negative numOpaqueSecrets is clamped and errors": {
			input:       Workload{SecretWorkload: SecretWorkload{NumOpaqueSecrets: -3}},
			expected:    Workload{SecretWorkload: SecretWorkload{NumOpaqueSecrets: 0}},
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			workload := tc.input

			err := validateWorkload(&workload)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, workload)
		})
	}
}
