package metrics

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsServerAddressEnvs(t *testing.T) {
	cases := map[string]struct {
		metricsPort         string
		enableSecureMetrics string
		secureMetricsPort   string
	}{
		"default": {
			metricsPort:         "",
			enableSecureMetrics: "false",
			secureMetricsPort:   "",
		},
		"only metricsPort set": {
			metricsPort:         ":8008",
			enableSecureMetrics: "false",
			secureMetricsPort:   "",
		},
		"only secureMetricsPort set": {
			metricsPort:         "",
			enableSecureMetrics: "true",
			secureMetricsPort:   ":8009",
		},
		"metrisPort and secureMetricsPort set": {
			metricsPort:         ":8008",
			enableSecureMetrics: "true",
			secureMetricsPort:   ":8009",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.MetricsPort.EnvVar(), c.metricsPort)
			t.Setenv(env.EnableSecureMetrics.EnvVar(), c.enableSecureMetrics)
			t.Setenv(env.SecureMetricsPort.EnvVar(), c.secureMetricsPort)

			server := NewServer(CentralSubsystem, &nilTLSConfigurer{})

			require.NotNil(t, server)
			assert.Equal(t, env.MetricsPort.Setting(), server.metricsServer.Addr)
			if c.enableSecureMetrics == "true" {
				require.NotNil(t, server.secureMetricsServer)
				assert.Equal(t, env.SecureMetricsPort.Setting(), server.secureMetricsServer.Addr)
			} else {
				assert.Nil(t, server.secureMetricsServer)
			}
		})
	}
}

func TestMetricsServerPanic(t *testing.T) {
	cases := map[string]struct {
		metricsPort         string
		enableSecureMetrics string
		secureMetricsPort   string
		releaseBuild        bool
	}{
		"metrics error - debug build panics": {
			metricsPort:         "error",
			enableSecureMetrics: "false",
			secureMetricsPort:   "",
			releaseBuild:        false,
		},
		"metrics error - release build does not panic": {
			metricsPort:         "error",
			enableSecureMetrics: "false",
			secureMetricsPort:   "",
			releaseBuild:        true,
		},
		"secureMetrics error - debug build panics": {
			metricsPort:         "",
			enableSecureMetrics: "true",
			secureMetricsPort:   "error",
			releaseBuild:        false,
		},
		"secureMetrics error - release build does not panic": {
			metricsPort:         "",
			enableSecureMetrics: "true",
			secureMetricsPort:   "error",
			releaseBuild:        true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if buildinfo.ReleaseBuild != c.releaseBuild {
				t.SkipNow()
			}
			t.Setenv(env.MetricsPort.EnvVar(), c.metricsPort)
			t.Setenv(env.EnableSecureMetrics.EnvVar(), c.enableSecureMetrics)
			t.Setenv(env.SecureMetricsPort.EnvVar(), c.secureMetricsPort)
			server := NewServer(CentralSubsystem, &nilTLSConfigurer{})
			defer server.Stop(context.TODO())

			if c.releaseBuild {
				assert.NotPanics(t, func() { server.RunForever() })
			} else {
				assert.Panics(t, func() { server.RunForever() })
			}
		})
	}
}
