package migratetooperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeSource struct {
	centralDep   *appsv1.Deployment
	centralDBDep *appsv1.Deployment
	err          error
}

func (f *fakeSource) CentralDeployment() (*appsv1.Deployment, error) {
	if f.centralDep == nil && f.err == nil {
		return makeCentralDeployment(nil), nil
	}
	return f.centralDep, f.err
}

func (f *fakeSource) CentralDBDeployment() (*appsv1.Deployment, error) {
	return f.centralDBDep, f.err
}

func makeCentralDBDeployment(volumes []corev1.Volume, nodeSelector map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "central-db"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes:      volumes,
					NodeSelector: nodeSelector,
				},
			},
		},
	}
}

func makeCentralDeployment(envVars []corev1.EnvVar) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "central"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "central",
						Env:  envVars,
					}},
				},
			},
		},
	}
}

func TestDetectStorage(t *testing.T) {
	tests := map[string]struct {
		volumes      []corev1.Volume
		nodeSelector map[string]string
		expected     storageConfig
	}{
		"PVC with default name": {
			volumes: []corev1.Volume{{
				Name: "disk",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "central-db",
					},
				},
			}},
			expected: storageConfig{Type: storagePVC, PVCName: "central-db"},
		},
		"PVC with custom name": {
			volumes: []corev1.Volume{{
				Name: "disk",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "my-custom-db",
					},
				},
			}},
			expected: storageConfig{Type: storagePVC, PVCName: "my-custom-db"},
		},
		"hostPath with default path": {
			volumes: []corev1.Volume{{
				Name: "disk",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/stackrox-central",
					},
				},
			}},
			expected: storageConfig{Type: storageHostPath, HostPath: "/var/lib/stackrox-central"},
		},
		"hostPath with custom path and nodeSelector": {
			volumes: []corev1.Volume{{
				Name: "disk",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/data/stackrox",
					},
				},
			}},
			nodeSelector: map[string]string{"kubernetes.io/hostname": "worker-1"},
			expected: storageConfig{
				Type:         storageHostPath,
				HostPath:     "/data/stackrox",
				NodeSelector: map[string]string{"kubernetes.io/hostname": "worker-1"},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			src := &fakeSource{centralDBDep: makeCentralDBDeployment(tt.volumes, tt.nodeSelector)}
			config, err := detect(src)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, config.Storage)
		})
	}
}

func TestDetectStorageErrors(t *testing.T) {
	tests := map[string]struct {
		volumes []corev1.Volume
		errMsg  string
	}{
		"no disk volume": {
			volumes: []corev1.Volume{{Name: "other"}},
			errMsg:  "no volume named \"disk\"",
		},
		"disk volume with neither PVC nor hostPath": {
			volumes: []corev1.Volume{{
				Name:         "disk",
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			}},
			errMsg: "neither a PVC nor a hostPath",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			src := &fakeSource{centralDBDep: makeCentralDBDeployment(tt.volumes, nil)}
			_, err := detect(src)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestDetectMonitoring(t *testing.T) {
	tests := map[string]struct {
		envVars           []corev1.EnvVar
		expectIsOpenShift bool
		expectMonitoring  bool
	}{
		"openshift with monitoring enabled": {
			envVars: []corev1.EnvVar{
				{Name: "ROX_ENABLE_OPENSHIFT_AUTH", Value: "true"},
				{Name: "ROX_ENABLE_SECURE_METRICS", Value: "true"},
			},
			expectIsOpenShift: true,
			expectMonitoring:  true,
		},
		"openshift with monitoring disabled": {
			envVars: []corev1.EnvVar{
				{Name: "ROX_ENABLE_OPENSHIFT_AUTH", Value: "true"},
			},
			expectIsOpenShift: true,
			expectMonitoring:  false,
		},
		"k8s (no openshift env vars)": {
			envVars:           nil,
			expectIsOpenShift: false,
			expectMonitoring:  false,
		},
		"k8s with other env vars": {
			envVars: []corev1.EnvVar{
				{Name: "ROX_OFFLINE_MODE", Value: "false"},
			},
			expectIsOpenShift: false,
			expectMonitoring:  false,
		},
	}

	pvcVolume := []corev1.Volume{{
		Name:         "disk",
		VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "central-db"}},
	}}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			src := &fakeSource{
				centralDep:   makeCentralDeployment(tt.envVars),
				centralDBDep: makeCentralDBDeployment(pvcVolume, nil),
			}
			config, err := detect(src)
			require.NoError(t, err)
			assert.Equal(t, tt.expectIsOpenShift, config.Monitoring.IsOpenShift)
			assert.Equal(t, tt.expectMonitoring, config.Monitoring.OpenShiftMonitoringEnabled)
		})
	}
}
