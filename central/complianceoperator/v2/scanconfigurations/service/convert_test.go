package service

import (
	"context"
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
)

// TestWriteNodeRoles verifies the write-path node-role handling. With the feature
// enabled: empty input gets the backward-compatible default and custom roles pass
// through. With the feature disabled: caller-supplied roles are dropped (nil
// stored) so the feature cannot be exercised through the API.
func TestWriteNodeRoles(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		t.Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
		testCases := map[string]struct {
			input    []string
			expected []string
		}{
			"empty defaults to master and worker":       {nil, []string{"master", "worker"}},
			"empty slice defaults to master and worker": {[]string{}, []string{"master", "worker"}},
			"custom roles pass through unchanged":       {[]string{"infra", "control-plane"}, []string{"infra", "control-plane"}},
			"@all passes through unchanged":             {[]string{allNodesRole}, []string{allNodesRole}},
		}
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				assert.Equal(t, tc.expected, writeNodeRoles(tc.input))
			})
		}
	})

	t.Run("disabled drops roles", func(t *testing.T) {
		t.Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "false")
		assert.Nil(t, writeNodeRoles(nil))
		assert.Nil(t, writeNodeRoles([]string{"infra"}))
		assert.Nil(t, writeNodeRoles([]string{allNodesRole}))
	})
}

// TestConvertV2ScanConfigToStorageNodeRoles exercises the full write-path
// conversion with the feature enabled and the new per-cluster node roles.
func TestConvertV2ScanConfigToStorageNodeRoles(t *testing.T) {
	t.Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
	testCases := map[string]struct {
		inputRoles map[string][]string // clusterID -> roles
		expected   map[string][]string // clusterID -> expected storage roles
	}{
		"empty defaults to master and worker": {
			inputRoles: map[string][]string{"cluster-1": nil},
			expected:   map[string][]string{"cluster-1": {"master", "worker"}},
		},
		"custom roles pass through unchanged": {
			inputRoles: map[string][]string{"cluster-1": {"infra", "control-plane"}},
			expected:   map[string][]string{"cluster-1": {"infra", "control-plane"}},
		},
		"@all passes through unchanged": {
			inputRoles: map[string][]string{"cluster-1": {allNodesRole}},
			expected:   map[string][]string{"cluster-1": {allNodesRole}},
		},
		"multiple clusters with different roles": {
			inputRoles: map[string][]string{
				"cluster-1": {"infra"},
				"cluster-2": nil,
				"cluster-3": {allNodesRole},
			},
			expected: map[string][]string{
				"cluster-1": {"infra"},
				"cluster-2": {"master", "worker"},
				"cluster-3": {allNodesRole},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			clusterIDs := make([]string, 0, len(tc.inputRoles))
			clusterNodeRoles := make(map[string]*v2.NodeRoleSet, len(tc.inputRoles))
			for clusterID, roles := range tc.inputRoles {
				clusterIDs = append(clusterIDs, clusterID)
				if roles != nil {
					clusterNodeRoles[clusterID] = &v2.NodeRoleSet{NodeRoles: roles}
				}
			}
			cfg := &v2.ComplianceScanConfiguration{
				ScanName:         "test-scan",
				Clusters:         clusterIDs,
				ClusterNodeRoles: clusterNodeRoles,
				ScanConfig:       &v2.BaseComplianceScanConfigurationSettings{},
			}
			storageCfg := convertV2ScanConfigToStorage(context.Background(), cfg)
			// Verify each cluster has the expected roles
			for _, cluster := range storageCfg.GetClusters() {
				expectedRoles, ok := tc.expected[cluster.GetClusterId()]
				assert.True(t, ok, "unexpected cluster %s in storage", cluster.GetClusterId())
				assert.Equal(t, expectedRoles, cluster.GetNodeRoles())
			}
		})
	}
}

// TestReadNodeRoles verifies the read-path helper used by the storage->v2
// converters. With the feature enabled, pre-PR stored configs (empty node_roles)
// display as the master+worker roles they actually run on Sensor. With the
// feature disabled, node roles are not surfaced on the API at all.
func TestReadNodeRoles(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		t.Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
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
				assert.Equal(t, tc.expected, readNodeRoles(tc.input))
			})
		}
	})

	t.Run("disabled hides roles", func(t *testing.T) {
		t.Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "false")
		assert.Nil(t, readNodeRoles(nil))
		assert.Nil(t, readNodeRoles([]string{"infra"}))
		assert.Nil(t, readNodeRoles([]string{allNodesRole}))
	})
}
