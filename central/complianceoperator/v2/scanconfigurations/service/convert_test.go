package service

import (
	"context"
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stretchr/testify/assert"
)

// TestConvertV2ScanConfigToStorageNodeRoles verifies the write-path node-role
// defaulting: empty input gets the backward-compatible default, custom roles
// pass through unchanged.
func TestConvertV2ScanConfigToStorageNodeRoles(t *testing.T) {
	testCases := map[string]struct {
		input    []string
		expected []string
	}{
		"empty defaults to master and worker": {
			input:    nil,
			expected: []string{"master", "worker"},
		},
		"empty slice defaults to master and worker": {
			input:    []string{},
			expected: []string{"master", "worker"},
		},
		"custom roles pass through unchanged": {
			input:    []string{"infra", "control-plane"},
			expected: []string{"infra", "control-plane"},
		},
		"@all passes through unchanged": {
			input:    []string{allNodesRole},
			expected: []string{allNodesRole},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			cfg := &v2.ComplianceScanConfiguration{
				ScanName: "test-scan",
				ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
					NodeRoles: tc.input,
				},
			}
			storageCfg := convertV2ScanConfigToStorage(context.Background(), cfg)
			assert.Equal(t, tc.expected, storageCfg.GetNodeRoles())
		})
	}
}

// TestNodeRolesOrDefault verifies the read-path defaulting helper used by the
// storage->v2 converters so pre-PR stored configs (empty node_roles) display as
// the master+worker roles they actually run on Sensor.
func TestNodeRolesOrDefault(t *testing.T) {
	testCases := map[string]struct {
		input    []string
		expected []string
	}{
		"nil defaults":   {nil, []string{"master", "worker"}},
		"empty defaults": {[]string{}, []string{"master", "worker"}},
		"custom passes":  {[]string{"infra"}, []string{"infra"}},
		"@all passes":    {[]string{allNodesRole}, []string{allNodesRole}},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, nodeRolesOrDefault(tc.input))
		})
	}
}

func TestDefaultNodeRoles(t *testing.T) {
	assert.Equal(t, []string{"master", "worker"}, defaultNodeRoles())
}
