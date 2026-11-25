package translation

import (
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestMonitoringValues(t *testing.T) {
	tests := map[string]struct {
		monitoring *platform.SecuredClusterMonitoring
		want       chartutil.Values
	}{
		"unset monitoring": {
			want: chartutil.Values{"openshift": map[string]interface{}{"enabled": true}},
		},
		"unset openshift": {
			monitoring: &platform.SecuredClusterMonitoring{},
			want:       chartutil.Values{"openshift": map[string]interface{}{"enabled": true}},
		},
		"unset flags": {
			monitoring: &platform.SecuredClusterMonitoring{
				OpenShiftMonitoring: &platform.SecuredClusterOpenShiftMonitoring{},
			},
			want: chartutil.Values{"openshift": map[string]interface{}{"enabled": true}},
		},
		"operational only": {
			monitoring: &platform.SecuredClusterMonitoring{
				OpenShiftMonitoring: &platform.SecuredClusterOpenShiftMonitoring{
					Enabled: new(true),
					Custom:  &platform.OpenShiftCustomMonitoring{Enabled: new(false)},
				},
			},
			want: chartutil.Values{"openshift": map[string]interface{}{"enabled": true}},
		},
		"custom only": {
			monitoring: &platform.SecuredClusterMonitoring{
				OpenShiftMonitoring: &platform.SecuredClusterOpenShiftMonitoring{
					Enabled: new(false),
					Custom:  &platform.OpenShiftCustomMonitoring{Enabled: new(true)},
				},
			},
			want: chartutil.Values{"openshift": map[string]interface{}{
				"enabled": false, "custom": map[string]interface{}{"enabled": true},
			}},
		},
		"both": {
			monitoring: &platform.SecuredClusterMonitoring{
				OpenShiftMonitoring: &platform.SecuredClusterOpenShiftMonitoring{
					Enabled: new(true),
					Custom:  &platform.OpenShiftCustomMonitoring{Enabled: new(true)},
				},
			},
			want: chartutil.Values{"openshift": map[string]interface{}{
				"enabled": true, "custom": map[string]interface{}{"enabled": true},
			}},
		},
		"custom with operational default": {
			monitoring: &platform.SecuredClusterMonitoring{
				OpenShiftMonitoring: &platform.SecuredClusterOpenShiftMonitoring{
					Custom: &platform.OpenShiftCustomMonitoring{Enabled: new(true)},
				},
			},
			want: chartutil.Values{"openshift": map[string]interface{}{
				"enabled": true, "custom": map[string]interface{}{"enabled": true},
			}},
		},
		"both disabled": {
			monitoring: &platform.SecuredClusterMonitoring{
				OpenShiftMonitoring: &platform.SecuredClusterOpenShiftMonitoring{
					Enabled: new(false),
					Custom:  &platform.OpenShiftCustomMonitoring{Enabled: new(false)},
				},
			},
			want: chartutil.Values{"openshift": map[string]interface{}{"enabled": false}},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := getMonitoringValues(tt.monitoring).Build()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
