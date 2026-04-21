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
	deployment *appsv1.Deployment
	err        error
}

func (f *fakeSource) CentralDBDeployment() (*appsv1.Deployment, error) {
	return f.deployment, f.err
}

func centralDBDeployment(volumes []corev1.Volume, nodeSelector map[string]string) *appsv1.Deployment {
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
			src := &fakeSource{deployment: centralDBDeployment(tt.volumes, tt.nodeSelector)}
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
			src := &fakeSource{deployment: centralDBDeployment(tt.volumes, nil)}
			_, err := detect(src)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}
