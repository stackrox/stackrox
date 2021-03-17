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
