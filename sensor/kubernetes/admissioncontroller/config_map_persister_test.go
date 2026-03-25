package admissioncontroller

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/gziputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestSettingsToConfigMap(t *testing.T) {
	cases := []struct {
		settings  *sensor.AdmissionControlSettings
		expectNil bool
	}{
		{
			settings:  nil,
			expectNil: true,
		},
		{
			settings:  &sensor.AdmissionControlSettings{},
			expectNil: true,
		},
		{
			settings: &sensor.AdmissionControlSettings{
				ClusterConfig: &storage.DynamicClusterConfig{},
			},
			expectNil: true,
		},
		{
			settings: &sensor.AdmissionControlSettings{
				EnforcedDeployTimePolicies: &storage.PolicyList{},
			},
			expectNil: true,
		},
		{
			settings: &sensor.AdmissionControlSettings{
				RuntimePolicies: &storage.PolicyList{},
			},
			expectNil: true,
		},
		{
			settings: &sensor.AdmissionControlSettings{
				ClusterConfig:              &storage.DynamicClusterConfig{},
				EnforcedDeployTimePolicies: &storage.PolicyList{},
			},
			expectNil: true,
		},
		{
			settings: &sensor.AdmissionControlSettings{
				ClusterConfig:   &storage.DynamicClusterConfig{},
				RuntimePolicies: &storage.PolicyList{},
			},
			expectNil: true,
		},
		{
			settings: &sensor.AdmissionControlSettings{
				ClusterConfig:              &storage.DynamicClusterConfig{},
				EnforcedDeployTimePolicies: &storage.PolicyList{},
				RuntimePolicies:            &storage.PolicyList{},
			},
			expectNil: false,
		},
	}

	for i, testCase := range cases {
		c := testCase
		t.Run(fmt.Sprintf("case#%d", i), func(t *testing.T) {
			var cm *v1.ConfigMap
			var err error
			require.NotPanics(t, func() {
				cm, err = settingsToConfigMap(c.settings)
			})
			require.NoError(t, err)

			if c.expectNil {
				assert.Nil(t, cm)
			} else {
				require.NotNil(t, cm)
				_, err := gziputil.Decompress(cm.BinaryData[admissioncontrol.ConfigGZDataKey])
				assert.NoError(t, err)
				_, err = gziputil.Decompress(cm.BinaryData[admissioncontrol.DeployTimePoliciesGZDataKey])
				assert.NoError(t, err)
				_, err = gziputil.Decompress(cm.BinaryData[admissioncontrol.RunTimePoliciesGZDataKey])
				assert.NoError(t, err)
			}
		})
	}
}

func TestSettingsToConfigMap_ClusterLabels(t *testing.T) {
	cases := []struct {
		name          string
		clusterLabels *sensor.ClusterLabels
		expectInMap   bool
	}{
		{
			name:          "nil cluster labels",
			clusterLabels: nil,
			expectInMap:   false,
		},
		{
			name:          "empty cluster labels",
			clusterLabels: &sensor.ClusterLabels{Labels: nil},
			expectInMap:   true,
		},
		{
			name: "cluster labels with data",
			clusterLabels: &sensor.ClusterLabels{
				Labels: map[string]string{
					"env":    "prod",
					"region": "us-east-1",
				},
			},
			expectInMap: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			settings := &sensor.AdmissionControlSettings{
				ClusterConfig:              &storage.DynamicClusterConfig{},
				EnforcedDeployTimePolicies: &storage.PolicyList{},
				RuntimePolicies:            &storage.PolicyList{},
				ClusterLabels:              c.clusterLabels,
			}

			cm, err := settingsToConfigMap(settings)
			require.NoError(t, err)
			require.NotNil(t, cm)

			if c.expectInMap {
				clusterLabelsData := cm.BinaryData[admissioncontrol.ClusterLabelsGZDataKey]
				require.NotNil(t, clusterLabelsData, "cluster labels should be in ConfigMap")

				// Verify it can be decompressed and unmarshaled
				decompressed, err := gziputil.Decompress(clusterLabelsData)
				require.NoError(t, err)

				var clusterLabels sensor.ClusterLabels
				err = clusterLabels.UnmarshalVT(decompressed)
				require.NoError(t, err)

				if c.clusterLabels.GetLabels() != nil {
					assert.Equal(t, c.clusterLabels.GetLabels(), clusterLabels.GetLabels())
				}
			} else {
				clusterLabelsData := cm.BinaryData[admissioncontrol.ClusterLabelsGZDataKey]
				assert.Nil(t, clusterLabelsData, "cluster labels should not be in ConfigMap when nil")
			}
		})
	}
}
