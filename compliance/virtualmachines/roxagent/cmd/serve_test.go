package cmd

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/assert"
)

func TestFormatDnfStatusFlags(t *testing.T) {
	tests := map[string]struct {
		flags    []v1.DnfStatusFlag
		expected string
	}{
		"nil flags": {
			flags:    nil,
			expected: "none",
		},
		"empty flags": {
			flags:    []v1.DnfStatusFlag{},
			expected: "none",
		},
		"single flag": {
			flags:    []v1.DnfStatusFlag{v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND},
			expected: "DNF_REPO_CONFIG_FOUND",
		},
		"multiple flags are sorted": {
			flags: []v1.DnfStatusFlag{
				v1.DnfStatusFlag_DNF_V4_HISTORY_DB_FOUND,
				v1.DnfStatusFlag_DNF_REPO_CONFIG_FOUND,
				v1.DnfStatusFlag_DNF_V4_CACHE_FOUND,
			},
			expected: "DNF_REPO_CONFIG_FOUND, DNF_V4_CACHE_FOUND, DNF_V4_HISTORY_DB_FOUND",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatDnfStatusFlags(tt.flags))
		})
	}
}
