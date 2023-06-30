package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {

	tc := []struct {
		name    string
		args    []string
		envVars map[string]string
		wantErr bool
		assert  func(t *testing.T)
	}{
		{
			name: "default",
			assert: func(t *testing.T) {
				assert.Equal(t, ":8080", metricsBindAddr)
				assert.Equal(t, ":8081", healthProbeBindAddrress)
				assert.Equal(t, "/readyz", readinessProbeEndpointName)
				assert.Equal(t, "/healthz", livenessProbeEndpointName)
				assert.Equal(t, false, enableLeaderElection)
				assert.Equal(t, 15*time.Second, leaderElectLeaseDuration)
				assert.Equal(t, 10*time.Second, leaderElectRenewDeadline)
				assert.Equal(t, 2*time.Second, leaderElectRetryPeriod)
				assert.Equal(t, "bf7ea6a2.stackrox.io", leaderElectID)
				assert.Equal(t, "", leaderElectNamespace)
				assert.Equal(t, 10*time.Hour, syncPeriod)
				assert.Equal(t, 9443, webhookPort)
				assert.Equal(t, "", webhookHost)
				assert.Equal(t, true, enableWebhooks)
				assert.Equal(t, false, enableProfiling)
				assert.Equal(t, 0.8, profilingThresholdFraction)
				assert.Equal(t, uint64(0), memLimit)
				assert.Equal(t, os.TempDir(), heapDumpDir)
				assert.Equal(t, 30*time.Second, gracefulShutdownTimeout)
				assert.Equal(t, false, dryRunClient)
				assert.Equal(t, "", centralLabelSelector)
				assert.Equal(t, true, enableCentralReconciler)
				assert.Equal(t, true, enableSecuredClusterReconciler)
			},
		}, {
			name:    "metricsBindAddr env var",
			envVars: map[string]string{"METRICS_BIND_ADDRESS": "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", metricsBindAddr)
			},
		}, {
			name: "metricsBindAddr flag",
			args: []string{"--metrics-bind-address", "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", metricsBindAddr)
			},
		}, {
			name:    "metricsBindAddr flag overrides env var",
			args:    []string{"--metrics-bind-address", "custom"},
			envVars: map[string]string{"METRICS_BIND_ADDRESS": "env"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", metricsBindAddr)
			},
		}, {
			name:    "healthProbeBindAddrress env var",
			envVars: map[string]string{"HEALTH_PROBE_BIND_ADDRESS": "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", healthProbeBindAddrress)
			},
		}, {
			name: "healthProbeBindAddrress flag",
			args: []string{"--health-probe-bind-address", "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", healthProbeBindAddrress)
			},
		}, {
			name:    "healthProbeBindAddrress flag overrides env var",
			args:    []string{"--health-probe-bind-address", "custom"},
			envVars: map[string]string{"HEALTH_PROBE_BIND_ADDRESS": "env"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", healthProbeBindAddrress)
			},
		}, {
			name:    "readinessProbeEndpointName env var",
			envVars: map[string]string{"READINESS_PROBE_ENDPOINT_NAME": "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", readinessProbeEndpointName)
			},
		}, {
			name: "readinessProbeEndpointName flag",
			args: []string{"--readiness-probe-endpoint-name", "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", readinessProbeEndpointName)
			},
		}, {
			name:    "readinessProbeEndpointName flag overrides env var",
			args:    []string{"--readiness-probe-endpoint-name", "custom"},
			envVars: map[string]string{"READINESS_PROBE_ENDPOINT_NAME": "env"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", readinessProbeEndpointName)
			},
		}, {
			name:    "livenessProbeEndpointName env var",
			envVars: map[string]string{"LIVENESS_PROBE_ENDPOINT_NAME": "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", livenessProbeEndpointName)
			},
		}, {
			name: "livenessProbeEndpointName flag",
			args: []string{"--liveness-probe-endpoint-name", "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", livenessProbeEndpointName)
			},
		}, {
			name:    "livenessProbeEndpointName flag overrides env var",
			args:    []string{"--liveness-probe-endpoint-name", "custom"},
			envVars: map[string]string{"LIVENESS_PROBE_ENDPOINT_NAME": "env"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", livenessProbeEndpointName)
			},
		}, {
			name:    "enableLeaderElection env var",
			envVars: map[string]string{"LEADER_ELECT": "true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableLeaderElection)
			},
		}, {
			name: "enableLeaderElection flag",
			args: []string{"--leader-elect=true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableLeaderElection)
			},
		}, {
			name: "enableLeaderElection flag present",
			args: []string{"--leader-elect"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableLeaderElection)
			},
		}, {
			name:    "enableLeaderElection flag overrides env var",
			args:    []string{"--leader-elect=true"},
			envVars: map[string]string{"ENABLE_LEADER_ELECTION": "false"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableLeaderElection)
			},
		}, {
			name:    "leaderElectLeaseDuration env var",
			envVars: map[string]string{"LEADER_ELECT_LEASE_DURATION": "1s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, leaderElectLeaseDuration)
			},
		}, {
			name: "leaderElectLeaseDuration flag",
			args: []string{"--leader-elect-lease-duration", "1s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, leaderElectLeaseDuration)
			},
		}, {
			name:    "leaderElectLeaseDuration flag overrides env var",
			args:    []string{"--leader-elect-lease-duration", "1s"},
			envVars: map[string]string{"LEADER_ELECT_LEASE_DURATION": "2s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, leaderElectLeaseDuration)
			},
		}, {
			name:    "leaderElectRenewDeadline env var",
			envVars: map[string]string{"LEADER_ELECT_RENEW_DEADLINE": "1s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, leaderElectRenewDeadline)
			},
		}, {
			name: "leaderElectRenewDeadline flag",
			args: []string{"--leader-elect-renew-deadline", "1s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, leaderElectRenewDeadline)
			},
		}, {
			name:    "leaderElectRenewDeadline flag overrides env var",
			args:    []string{"--leader-elect-renew-deadline", "1s"},
			envVars: map[string]string{"LEADER_ELECT_RENEW_DEADLINE": "2s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, leaderElectRenewDeadline)
			},
		}, {
			name:    "leaderElectRetryPeriod env var",
			envVars: map[string]string{"LEADER_ELECT_RETRY_PERIOD": "1s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, leaderElectRetryPeriod)
			},
		}, {
			name: "leaderElectRetryPeriod flag",
			args: []string{"--leader-elect-retry-period", "1s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, leaderElectRetryPeriod)
			},
		}, {
			name:    "leaderElectRetryPeriod flag overrides env var",
			args:    []string{"--leader-elect-retry-period", "1s"},
			envVars: map[string]string{"LEADER_ELECT_RETRY_PERIOD": "2s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, leaderElectRetryPeriod)
			},
		}, {
			name:    "leaderElectID env var",
			envVars: map[string]string{"LEADER_ELECT_ID": "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", leaderElectID)
			},
		}, {
			name: "leaderElectID flag",
			args: []string{"--leader-elect-id", "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", leaderElectID)
			},
		}, {
			name:    "leaderElectID flag overrides env var",
			args:    []string{"--leader-elect-id", "custom"},
			envVars: map[string]string{"LEADER_ELECT_ID": "env"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", leaderElectID)
			},
		}, {
			name:    "leaderElectNamespace env var",
			envVars: map[string]string{"LEADER_ELECT_NAMESPACE": "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", leaderElectNamespace)
			},
		}, {
			name: "leaderElectNamespace flag",
			args: []string{"--leader-elect-namespace", "custom"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", leaderElectNamespace)
			},
		}, {
			name:    "leaderElectNamespace flag overrides env var",
			args:    []string{"--leader-elect-namespace", "custom"},
			envVars: map[string]string{"LEADER_ELECT_NAMESPACE": "env"},
			assert: func(t *testing.T) {
				assert.Equal(t, "custom", leaderElectNamespace)
			},
		}, {
			name:    "syncPeriod env var",
			envVars: map[string]string{"SYNC_PERIOD": "1s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, syncPeriod)
			},
		}, {
			name: "syncPeriod flag",
			args: []string{"--sync-period", "1s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, syncPeriod)
			},
		}, {
			name:    "syncPeriod flag overrides env var",
			args:    []string{"--sync-period", "1s"},
			envVars: map[string]string{"SYNC_PERIOD": "2s"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Second, syncPeriod)
			},
		}, {
			name:    "webhookPort env var",
			envVars: map[string]string{"WEBHOOK_BIND_PORT": "8080"},
			assert: func(t *testing.T) {
				assert.Equal(t, 8080, webhookPort)
			},
		}, {
			name: "webhookPort flag",
			args: []string{"--webhook-bind-port", "8080"},
			assert: func(t *testing.T) {
				assert.Equal(t, 8080, webhookPort)
			},
		}, {
			name:    "webhookPort flag overrides env var",
			args:    []string{"--webhook-bind-port", "8080"},
			envVars: map[string]string{"WEBHOOK_BIND_PORT": "9090"},
			assert: func(t *testing.T) {
				assert.Equal(t, 8080, webhookPort)
			},
		}, {
			name:    "webhookHost env var",
			envVars: map[string]string{"WEBHOOK_BIND_HOST": "example.com"},
			assert: func(t *testing.T) {
				assert.Equal(t, "example.com", webhookHost)
			},
		}, {
			name: "webhookHost flag",
			args: []string{"--webhook-bind-host", "example.com"},
			assert: func(t *testing.T) {
				assert.Equal(t, "example.com", webhookHost)
			},
		}, {
			name:    "webhookHost flag overrides env var",
			args:    []string{"--webhook-bind-host", "example.com"},
			envVars: map[string]string{"WEBHOOK_BIND_HOST": "example.org"},
			assert: func(t *testing.T) {
				assert.Equal(t, "example.com", webhookHost)
			},
		}, {
			name:    "enableWebhooks env var",
			envVars: map[string]string{"ENABLE_WEBHOOKS": "true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableWebhooks)
			},
		}, {
			name: "enableWebhooks flag",
			args: []string{"--enable-webhooks=true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableWebhooks)
			},
		}, {
			name: "enableWebhooks flag present",
			args: []string{"--enable-webhooks"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableWebhooks)
			},
		}, {
			name:    "enableWebhooks flag overrides env var",
			args:    []string{"--enable-webhooks=true"},
			envVars: map[string]string{"ENABLE_WEBHOOKS": "false"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableWebhooks)
			},
		}, {
			name:    "enableProfiling env var",
			envVars: map[string]string{"ENABLE_PROFILING": "true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableProfiling)
			},
		}, {
			name: "enableProfiling flag",
			args: []string{"--enable-profiling=true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableProfiling)
			},
		}, {
			name: "enableProfiling flag present",
			args: []string{"--enable-profiling"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableProfiling)
			},
		}, {
			name:    "enableProfiling flag overrides env var",
			args:    []string{"--enable-profiling=true"},
			envVars: map[string]string{"ENABLE_PROFILING": "false"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableProfiling)
			},
		}, {
			name:    "profilingThresholdFraction env var",
			envVars: map[string]string{"PROFILING_THRESHOLD_FRACTION": "0.5"},
			assert: func(t *testing.T) {
				assert.Equal(t, 0.5, profilingThresholdFraction)
			},
		}, {
			name: "profilingThresholdFraction flag",
			args: []string{"--profiling-threshold-fraction", "0.5"},
			assert: func(t *testing.T) {
				assert.Equal(t, 0.5, profilingThresholdFraction)
			},
		}, {
			name:    "profilingThresholdFraction flag overrides env var",
			args:    []string{"--profiling-threshold-fraction", "0.5"},
			envVars: map[string]string{"PROFILING_THRESHOLD_FRACTION": "0.6"},
			assert: func(t *testing.T) {
				assert.Equal(t, 0.5, profilingThresholdFraction)
			},
		}, {
			name:    "memLimit env var",
			envVars: map[string]string{"MEMORY_LIMIT_BYTES": "1000000"},
			assert: func(t *testing.T) {
				assert.Equal(t, uint64(1000000), memLimit)
			},
		}, {
			name: "memLimit flag",
			args: []string{"--memory-limit-bytes", "1000000"},
			assert: func(t *testing.T) {
				assert.Equal(t, uint64(1000000), memLimit)
			},
		}, {
			name:    "memLimit flag overrides env var",
			args:    []string{"--memory-limit-bytes", "1000000"},
			envVars: map[string]string{"MEMORY_LIMIT_BYTES": "2000000"},
			assert: func(t *testing.T) {
				assert.Equal(t, uint64(1000000), memLimit)
			},
		}, {
			name:    "heapDumpDir env var",
			envVars: map[string]string{"HEAP_DUMP_PARENT_DIR": "/tmp"},
			assert: func(t *testing.T) {
				assert.Equal(t, "/tmp", heapDumpDir)
			},
		}, {
			name: "heapDumpDir flag",
			args: []string{"--heap-dump-parent-dir", "/tmp"},
			assert: func(t *testing.T) {
				assert.Equal(t, "/tmp", heapDumpDir)
			},
		}, {
			name:    "heapDumpDir flag overrides env var",
			args:    []string{"--heap-dump-parent-dir", "/tmp"},
			envVars: map[string]string{"HEAP_DUMP_PARENT_DIR": "/var"},
			assert: func(t *testing.T) {
				assert.Equal(t, "/tmp", heapDumpDir)
			},
		}, {
			name:    "gracefulShutdownTimeout env var",
			envVars: map[string]string{"GRACEFUL_SHUTDOWN_TIMEOUT": "1m"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Minute, gracefulShutdownTimeout)
			},
		}, {
			name: "gracefulShutdownTimeout flag",
			args: []string{"--graceful-shutdown-timeout", "1m"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Minute, gracefulShutdownTimeout)
			},
		}, {
			name:    "gracefulShutdownTimeout flag overrides env var",
			args:    []string{"--graceful-shutdown-timeout", "1m"},
			envVars: map[string]string{"GRACEFUL_SHUTDOWN_TIMEOUT": "2m"},
			assert: func(t *testing.T) {
				assert.Equal(t, time.Minute, gracefulShutdownTimeout)
			},
		}, {
			name:    "dryRunClient env var",
			envVars: map[string]string{"DRY_RUN_CLIENT": "true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, dryRunClient)
			},
		}, {
			name: "dryRunClient flag",
			args: []string{"--dry-run-client=true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, dryRunClient)
			},
		}, {
			name: "dryRunClient flag present",
			args: []string{"--dry-run-client"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, dryRunClient)
			},
		}, {
			name:    "dryRunClient flag overrides env var",
			args:    []string{"--dry-run-client=true"},
			envVars: map[string]string{"DRY_RUN_CLIENT": "false"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, dryRunClient)
			},
		}, {
			name:    "centralLabelSelector env var",
			envVars: map[string]string{"CENTRAL_LABEL_SELECTOR": "foo=bar"},
			assert: func(t *testing.T) {
				assert.Equal(t, "foo=bar", centralLabelSelector)
			},
		}, {
			name: "centralLabelSelector flag",
			args: []string{"--central-label-selector", "foo=bar"},
			assert: func(t *testing.T) {
				assert.Equal(t, "foo=bar", centralLabelSelector)
			},
		}, {
			name:    "centralLabelSelector flag overrides env var",
			args:    []string{"--central-label-selector", "foo=bar"},
			envVars: map[string]string{"CENTRAL_LABEL_SELECTOR": "bar=baz"},
			assert: func(t *testing.T) {
				assert.Equal(t, "foo=bar", centralLabelSelector)
			},
		}, {
			name:    "enableCentralReconciler env var",
			envVars: map[string]string{"ENABLE_CENTRAL_RECONCILER": "false"},
			assert: func(t *testing.T) {
				assert.Equal(t, false, enableCentralReconciler)
			},
		}, {
			name: "enableCentralReconciler flag",
			args: []string{"--enable-central-reconciler=false"},
			assert: func(t *testing.T) {
				assert.Equal(t, false, enableCentralReconciler)
			},
		}, {
			name: "enableCentralReconciler flag present",
			args: []string{"--enable-central-reconciler"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableCentralReconciler)
			},
		}, {
			name:    "enableCentralReconciler flag overrides env var",
			args:    []string{"--enable-central-reconciler=false"},
			envVars: map[string]string{"ENABLE_CENTRAL_RECONCILER": "true"},
			assert: func(t *testing.T) {
				assert.Equal(t, false, enableCentralReconciler)
			},
		}, {
			name:    "enableSecuredClusterReconciler env var",
			envVars: map[string]string{"ENABLE_SECURED_CLUSTER_RECONCILER": "false"},
			assert: func(t *testing.T) {
				assert.Equal(t, false, enableSecuredClusterReconciler)
			},
		}, {
			name: "enableSecuredClusterReconciler flag",
			args: []string{"--enable-secured-cluster-reconciler=false"},
			assert: func(t *testing.T) {
				assert.Equal(t, false, enableSecuredClusterReconciler)
			},
		}, {
			name: "enableSecuredClusterReconciler flag present",
			args: []string{"--enable-secured-cluster-reconciler"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, enableSecuredClusterReconciler)
			},
		}, {
			name:    "enableSecuredClusterReconciler flag overrides env var",
			args:    []string{"--enable-secured-cluster-reconciler=false"},
			envVars: map[string]string{"ENABLE_SECURED_CLUSTER_RECONCILER": "true"},
			assert: func(t *testing.T) {
				assert.Equal(t, false, enableSecuredClusterReconciler)
			},
		}, {
			name: "zap flag",
			args: []string{"--zap-devel=true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, zapOptions.Development)
			},
		}, {
			name: "zap flag present",
			args: []string{"--zap-devel"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, zapOptions.Development)
			},
		}, {
			name:    "zap env var",
			envVars: map[string]string{"ZAP_DEVEL": "true"},
			assert: func(t *testing.T) {
				assert.Equal(t, true, zapOptions.Development)
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			for key, envVar := range tt.envVars {
				t.Setenv(key, envVar)
			}
			startCmd.RunE = func(cmd *cobra.Command, args []string) error {
				return nil
			}
			rootCmd.SetArgs(append([]string{"start"}, tt.args...))
			err := rootCmd.Execute()
			require.NoError(t, err)
			tt.assert(t)
		})
	}

}
