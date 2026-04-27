package migratetooperator

import (
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestGenerateCR_PVC(t *testing.T) {
	config := &centralConfig{
		storage: storageConfig{
			typ:     storagePVC,
			pvcName: "my-db-pvc",
		},
	}

	cr, _ := generateCentral(config)

	assert.Equal(t, "platform.stackrox.io/v1alpha1", cr.APIVersion)
	assert.Equal(t, "Central", cr.Kind)
	assert.Equal(t, "stackrox-central-services", cr.Name)
	require.NotNil(t, cr.Spec.Central)
	require.NotNil(t, cr.Spec.Central.DB)
	require.NotNil(t, cr.Spec.Central.DB.Persistence)
	require.NotNil(t, cr.Spec.Central.DB.Persistence.PersistentVolumeClaim)
	assert.Equal(t, "my-db-pvc", *cr.Spec.Central.DB.Persistence.PersistentVolumeClaim.ClaimName)
	assert.Nil(t, cr.Spec.Central.DB.Persistence.HostPath)

	out, err := yaml.Marshal(cr)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "hostPath")
	assert.NotContains(t, string(out), "nodeSelector")
}

func TestGenerateCR_HostPath(t *testing.T) {
	config := &centralConfig{
		storage: storageConfig{
			typ:          storageHostPath,
			hostPath:     "/data/stackrox",
			nodeSelector: map[string]string{"kubernetes.io/hostname": "worker-1"},
		},
	}

	cr, _ := generateCentral(config)

	require.NotNil(t, cr.Spec.Central.DB.Persistence.HostPath)
	assert.Equal(t, "/data/stackrox", *cr.Spec.Central.DB.Persistence.HostPath.Path)
	assert.Nil(t, cr.Spec.Central.DB.Persistence.PersistentVolumeClaim)
	assert.Equal(t, "worker-1", cr.Spec.Central.DB.NodeSelector["kubernetes.io/hostname"])
}

func TestGenerateCR_HostPathWithoutNodeSelector(t *testing.T) {
	config := &centralConfig{
		storage: storageConfig{
			typ:      storageHostPath,
			hostPath: "/var/lib/stackrox-central",
		},
	}

	cr, _ := generateCentral(config)

	require.NotNil(t, cr.Spec.Central.DB.Persistence.HostPath)
	assert.Equal(t, "/var/lib/stackrox-central", *cr.Spec.Central.DB.Persistence.HostPath.Path)
	assert.Empty(t, cr.Spec.Central.DB.NodeSelector)
}

func TestGenerateCR_OpenShiftMonitoringEnabled(t *testing.T) {
	config := &centralConfig{
		storage:    storageConfig{typ: storagePVC, pvcName: "central-db"},
		monitoring: monitoringConfig{isOpenShift: true, openShiftMonitoringEnabled: true},
	}

	cr, _ := generateCentral(config)

	assert.Nil(t, cr.Spec.Monitoring)
}

func TestGenerateCR_OpenShiftMonitoringDisabled(t *testing.T) {
	config := &centralConfig{
		storage:    storageConfig{typ: storagePVC, pvcName: "central-db"},
		monitoring: monitoringConfig{isOpenShift: true, openShiftMonitoringEnabled: false},
	}

	cr, _ := generateCentral(config)

	require.NotNil(t, cr.Spec.Monitoring)
	require.NotNil(t, cr.Spec.Monitoring.OpenShiftMonitoring)
	require.NotNil(t, cr.Spec.Monitoring.OpenShiftMonitoring.Enabled)
	assert.False(t, *cr.Spec.Monitoring.OpenShiftMonitoring.Enabled)
}

func TestGenerateCR_K8sOmitsMonitoring(t *testing.T) {
	config := &centralConfig{
		storage:    storageConfig{typ: storagePVC, pvcName: "central-db"},
		monitoring: monitoringConfig{isOpenShift: false, openShiftMonitoringEnabled: false},
	}

	cr, _ := generateCentral(config)

	assert.Nil(t, cr.Spec.Monitoring)
}

func TestGenerateCR_ExposureLoadBalancer(t *testing.T) {
	config := &centralConfig{
		storage:  storageConfig{typ: storagePVC, pvcName: "central-db"},
		exposure: exposureConfig{loadBalancerEnabled: true},
	}
	cr, _ := generateCentral(config)
	require.NotNil(t, cr.Spec.Central.Exposure)
	require.NotNil(t, cr.Spec.Central.Exposure.LoadBalancer)
	assert.True(t, *cr.Spec.Central.Exposure.LoadBalancer.Enabled)
	assert.Nil(t, cr.Spec.Central.Exposure.NodePort)
	assert.Nil(t, cr.Spec.Central.Exposure.Route)
}

func TestGenerateCR_ExposureRoute(t *testing.T) {
	config := &centralConfig{
		storage:  storageConfig{typ: storagePVC, pvcName: "central-db"},
		exposure: exposureConfig{routeEnabled: true},
	}
	cr, _ := generateCentral(config)
	require.NotNil(t, cr.Spec.Central.Exposure)
	require.NotNil(t, cr.Spec.Central.Exposure.Route)
	assert.True(t, *cr.Spec.Central.Exposure.Route.Enabled)
}

func TestGenerateCR_ExposureNone(t *testing.T) {
	config := &centralConfig{
		storage: storageConfig{typ: storagePVC, pvcName: "central-db"},
	}
	cr, _ := generateCentral(config)
	assert.Nil(t, cr.Spec.Central.Exposure)
}

func TestGenerateCR_ExposureMultiple(t *testing.T) {
	config := &centralConfig{
		storage:  storageConfig{typ: storagePVC, pvcName: "central-db"},
		exposure: exposureConfig{loadBalancerEnabled: true, routeEnabled: true},
	}
	cr, _ := generateCentral(config)
	require.NotNil(t, cr.Spec.Central.Exposure)
	assert.True(t, *cr.Spec.Central.Exposure.LoadBalancer.Enabled)
	assert.True(t, *cr.Spec.Central.Exposure.Route.Enabled)
	assert.Nil(t, cr.Spec.Central.Exposure.NodePort)
}

func TestGenerateCR_OfflineMode(t *testing.T) {
	config := &centralConfig{
		storage:     storageConfig{typ: storagePVC, pvcName: "central-db"},
		offlineMode: true,
	}
	cr, _ := generateCentral(config)
	require.NotNil(t, cr.Spec.Egress)
	require.NotNil(t, cr.Spec.Egress.ConnectivityPolicy)
	assert.Equal(t, platform.ConnectivityOffline, *cr.Spec.Egress.ConnectivityPolicy)
}

func TestGenerateCR_OnlineMode(t *testing.T) {
	config := &centralConfig{
		storage: storageConfig{typ: storagePVC, pvcName: "central-db"},
	}
	cr, _ := generateCentral(config)
	assert.Nil(t, cr.Spec.Egress)
}

func TestGenerateCR_TelemetryDisabled(t *testing.T) {
	config := &centralConfig{
		storage:           storageConfig{typ: storagePVC, pvcName: "central-db"},
		telemetryDisabled: true,
	}
	cr, _ := generateCentral(config)
	require.NotNil(t, cr.Spec.Central.Telemetry)
	assert.False(t, *cr.Spec.Central.Telemetry.Enabled)
}

func TestGenerateCR_TelemetryDefault(t *testing.T) {
	config := &centralConfig{
		storage: storageConfig{typ: storagePVC, pvcName: "central-db"},
	}
	cr, _ := generateCentral(config)
	assert.Nil(t, cr.Spec.Central.Telemetry)
}

func TestGenerateCR_DefaultTLSSecret(t *testing.T) {
	config := &centralConfig{
		storage:              storageConfig{typ: storagePVC, pvcName: "central-db"},
		defaultTLSSecretName: "central-default-tls-cert",
	}
	cr, _ := generateCentral(config)
	require.NotNil(t, cr.Spec.Central.DefaultTLSSecret)
	assert.Equal(t, "central-default-tls-cert", cr.Spec.Central.DefaultTLSSecret.Name)
}

func TestGenerateCR_NoDefaultTLSSecret(t *testing.T) {
	config := &centralConfig{
		storage: storageConfig{typ: storagePVC, pvcName: "central-db"},
	}
	cr, _ := generateCentral(config)
	assert.Nil(t, cr.Spec.Central.DefaultTLSSecret)
}

func TestGenerateCR_DeclarativeConfig(t *testing.T) {
	config := &centralConfig{
		storage:               storageConfig{typ: storagePVC, pvcName: "central-db"},
		declarativeConfigMaps: []string{"my-cm"},
		declarativeSecrets:    []string{"my-secret"},
	}
	cr, _ := generateCentral(config)
	require.NotNil(t, cr.Spec.Central.DeclarativeConfiguration)
	require.Len(t, cr.Spec.Central.DeclarativeConfiguration.ConfigMaps, 1)
	assert.Equal(t, "my-cm", cr.Spec.Central.DeclarativeConfiguration.ConfigMaps[0].Name)
	require.Len(t, cr.Spec.Central.DeclarativeConfiguration.Secrets, 1)
	assert.Equal(t, "my-secret", cr.Spec.Central.DeclarativeConfiguration.Secrets[0].Name)
}

func TestGenerateCR_NoDeclarativeConfig(t *testing.T) {
	config := &centralConfig{
		storage: storageConfig{typ: storagePVC, pvcName: "central-db"},
	}
	cr, _ := generateCentral(config)
	assert.Nil(t, cr.Spec.Central.DeclarativeConfiguration)
}

func TestGenerateCR_PlaintextEndpoints(t *testing.T) {
	config := &centralConfig{
		storage:            storageConfig{typ: storagePVC, pvcName: "central-db"},
		plaintextEndpoints: "8080,grpc@8081",
	}
	cr, _ := generateCentral(config)
	require.NotNil(t, cr.Spec.Customize)
	require.Len(t, cr.Spec.Customize.EnvVars, 1)
	assert.Equal(t, "ROX_PLAINTEXT_ENDPOINTS", cr.Spec.Customize.EnvVars[0].Name)
	assert.Equal(t, "8080,grpc@8081", cr.Spec.Customize.EnvVars[0].Value)
}

func TestGenerateCR_NoPlaintextEndpoints(t *testing.T) {
	config := &centralConfig{
		storage: storageConfig{typ: storagePVC, pvcName: "central-db"},
	}
	cr, _ := generateCentral(config)
	assert.Nil(t, cr.Spec.Customize)
}
