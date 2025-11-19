package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestGlobalMonitoring_IsOpenShiftMonitoringDisabled(t *testing.T) {
	tests := map[string]struct {
		m    *GlobalMonitoring
		want bool
	}{
		"nil GlobalMonitoring": {
			m:    nil,
			want: false,
		},
		"empty GlobalMonitoring": {
			m:    &GlobalMonitoring{},
			want: false,
		},
		"nil OpenShiftMonitoring": {
			m: &GlobalMonitoring{
				OpenShiftMonitoring: nil,
			},
			want: false,
		},
		"empty OpenShiftMonitoring": {
			m: &GlobalMonitoring{
				OpenShiftMonitoring: &OpenShiftMonitoring{},
			},
			want: false,
		},
		"nil Enabled field": {
			m: &GlobalMonitoring{
				OpenShiftMonitoring: &OpenShiftMonitoring{
					Enabled: nil,
				},
			},
			want: false,
		},
		"Enabled true": {
			m: &GlobalMonitoring{
				OpenShiftMonitoring: &OpenShiftMonitoring{
					Enabled: ptr.To(true),
				},
			},
			want: false,
		},
		"Enabled false": {
			m: &GlobalMonitoring{
				OpenShiftMonitoring: &OpenShiftMonitoring{
					Enabled: ptr.To(false),
				},
			},
			want: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.m.IsOpenShiftMonitoringDisabled())
		})
	}
}
