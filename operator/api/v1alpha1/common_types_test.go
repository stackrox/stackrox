package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
					Enabled: new(true),
				},
			},
			want: false,
		},
		"Enabled false": {
			m: &GlobalMonitoring{
				OpenShiftMonitoring: &OpenShiftMonitoring{
					Enabled: new(false),
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

func TestGlobalMonitoring_IsOpenShiftCustomMonitoringEnabled(t *testing.T) {
	tests := map[string]struct {
		monitoring *GlobalMonitoring
		want       bool
	}{
		"nil monitoring": {},
		"nil openshift":  {monitoring: &GlobalMonitoring{}},
		"nil custom": {
			monitoring: &GlobalMonitoring{OpenShiftMonitoring: &OpenShiftMonitoring{}},
		},
		"nil enabled": {
			monitoring: &GlobalMonitoring{OpenShiftMonitoring: &OpenShiftMonitoring{
				Custom: &OpenShiftCustomMonitoring{},
			}},
		},
		"disabled": {
			monitoring: &GlobalMonitoring{OpenShiftMonitoring: &OpenShiftMonitoring{
				Custom: &OpenShiftCustomMonitoring{Enabled: new(false)},
			}},
		},
		"enabled": {
			monitoring: &GlobalMonitoring{OpenShiftMonitoring: &OpenShiftMonitoring{
				Custom: &OpenShiftCustomMonitoring{Enabled: new(true)},
			}},
			want: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.monitoring.IsOpenShiftCustomMonitoringEnabled())
		})
	}
}
