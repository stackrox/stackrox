package metrics

import (
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsServerAddressEnvs(t *testing.T) {
	cases := map[string]struct {
		metricsPort       string
		secureMetricsPort string
	}{
		"default": {
			metricsPort:       "",
			secureMetricsPort: "",
		},
		"only metricsPort set": {
			metricsPort:       ":8008",
			secureMetricsPort: "",
		},
		"only secureMetricsPort set": {
			metricsPort:       "",
			secureMetricsPort: ":8009",
		},
		"metrisPort and secureMetricsPort set": {
			metricsPort:       "8008",
			secureMetricsPort: ":8009",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.MetricsPort.EnvVar(), c.metricsPort)
			t.Setenv(env.SecureMetricsPort.EnvVar(), c.secureMetricsPort)

			server := NewMetricsServer(CentralSubsystem)

			require.NotNil(t, server)
			assert.Equal(t, env.MetricsPort.Setting(), server.Address)
			assert.Equal(t, env.SecureMetricsPort.Setting(), server.SecureAddress)
		})
	}
}

func TestMetricsServerPanic(t *testing.T) {
	cases := map[string]struct {
		metricsPort       string
		secureMetricsPort string
		releaseBuild      bool
	}{
		"metrics error - debug build panics": {
			metricsPort:       "error",
			secureMetricsPort: "",
			releaseBuild:      false,
		},
		"metrics error - release build does not panic": {
			metricsPort:       "error",
			secureMetricsPort: "",
			releaseBuild:      true,
		},
		"secureMetrics error - debug build panics": {
			metricsPort:       "",
			secureMetricsPort: "error",
			releaseBuild:      false,
		},
		"secureMetrics error - release build does not panic": {
			metricsPort:       "",
			secureMetricsPort: "error",
			releaseBuild:      true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if buildinfo.ReleaseBuild != c.releaseBuild {
				t.SkipNow()
			}
			t.Setenv(env.MetricsPort.EnvVar(), c.metricsPort)
			t.Setenv(env.SecureMetricsPort.EnvVar(), c.secureMetricsPort)

			if c.releaseBuild {
				assert.NotPanics(t, func() { NewMetricsServer(CentralSubsystem).RunForever() })
			} else {
				assert.Panics(t, func() { NewMetricsServer(CentralSubsystem).RunForever() })
			}
		})
	}
}
