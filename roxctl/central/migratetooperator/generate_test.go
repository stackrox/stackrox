package migratetooperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestGenerateCR_PVC(t *testing.T) {
	config := &detectedConfig{
		Storage: storageConfig{
			Type:    storagePVC,
			PVCName: "my-db-pvc",
		},
	}

	cr := generateCR(config)

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
	config := &detectedConfig{
		Storage: storageConfig{
			Type:         storageHostPath,
			HostPath:     "/data/stackrox",
			NodeSelector: map[string]string{"kubernetes.io/hostname": "worker-1"},
		},
	}

	cr := generateCR(config)

	require.NotNil(t, cr.Spec.Central.DB.Persistence.HostPath)
	assert.Equal(t, "/data/stackrox", *cr.Spec.Central.DB.Persistence.HostPath.Path)
	assert.Nil(t, cr.Spec.Central.DB.Persistence.PersistentVolumeClaim)
	assert.Equal(t, "worker-1", cr.Spec.Central.DB.NodeSelector["kubernetes.io/hostname"])
}

func TestGenerateCR_HostPathWithoutNodeSelector(t *testing.T) {
	config := &detectedConfig{
		Storage: storageConfig{
			Type:     storageHostPath,
			HostPath: "/var/lib/stackrox-central",
		},
	}

	cr := generateCR(config)

	require.NotNil(t, cr.Spec.Central.DB.Persistence.HostPath)
	assert.Equal(t, "/var/lib/stackrox-central", *cr.Spec.Central.DB.Persistence.HostPath.Path)
	assert.Empty(t, cr.Spec.Central.DB.NodeSelector)
}

func TestGenerateCR_OpenShiftMonitoringEnabled(t *testing.T) {
	config := &detectedConfig{
		Storage:    storageConfig{Type: storagePVC, PVCName: "central-db"},
		Monitoring: monitoringConfig{IsOpenShift: true, OpenShiftMonitoringEnabled: true},
	}

	cr := generateCR(config)

	assert.Nil(t, cr.Spec.Monitoring)
}

func TestGenerateCR_OpenShiftMonitoringDisabled(t *testing.T) {
	config := &detectedConfig{
		Storage:    storageConfig{Type: storagePVC, PVCName: "central-db"},
		Monitoring: monitoringConfig{IsOpenShift: true, OpenShiftMonitoringEnabled: false},
	}

	cr := generateCR(config)

	require.NotNil(t, cr.Spec.Monitoring)
	require.NotNil(t, cr.Spec.Monitoring.OpenShiftMonitoring)
	require.NotNil(t, cr.Spec.Monitoring.OpenShiftMonitoring.Enabled)
	assert.False(t, *cr.Spec.Monitoring.OpenShiftMonitoring.Enabled)
}

func TestGenerateCR_K8sOmitsMonitoring(t *testing.T) {
	config := &detectedConfig{
		Storage:    storageConfig{Type: storagePVC, PVCName: "central-db"},
		Monitoring: monitoringConfig{IsOpenShift: false, OpenShiftMonitoringEnabled: false},
	}

	cr := generateCR(config)

	assert.Nil(t, cr.Spec.Monitoring)
}
