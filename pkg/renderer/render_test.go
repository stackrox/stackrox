package renderer

import (
	"fmt"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/zip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRenderTLSSecretsOnly(t *testing.T) {
	config := Config{
		SecretsByteMap: map[string][]byte{
			"ca.pem":              []byte("CA"),
			"ca-key.pem":          []byte("CAKey"),
			"cert.pem":            []byte("CentralCert"),
			"key.pem":             []byte("CentralKey"),
			"scanner-cert.pem":    []byte("ScannerCert"),
			"scanner-key.pem":     []byte("ScannerKey"),
			"scanner-db-cert.pem": []byte("ScannerDBCert"),
			"scanner-db-key.pem":  []byte("ScannerDBKey"),
			"jwt-key.pem":         []byte("JWTKey"),
		},
		K8sConfig: &K8sConfig{
			DeploymentFormat: v1.DeploymentFormat_KUBECTL,
		},
	}

	for _, renderMode := range []mode{centralTLSOnly, scannerTLSOnly} {
		t.Run(fmt.Sprintf("mode=%s", renderMode), func(t *testing.T) {
			contents, err := renderAndExtractSingleFileContents(config, renderMode, testutils.MakeImageFlavorForTest(t))
			assert.NoError(t, err)

			objs, err := k8sutil.UnstructuredFromYAMLMulti(string(contents))
			assert.NoError(t, err)

			assert.NotEmpty(t, objs)
		})
	}
}

func TestRenderScannerOnly(t *testing.T) {
	flavor := testutils.MakeImageFlavorForTest(t)
	config := Config{
		SecretsByteMap: map[string][]byte{
			"ca.pem":              []byte("CA"),
			"ca-key.pem":          []byte("CAKey"),
			"cert.pem":            []byte("CentralCert"),
			"key.pem":             []byte("CentralKey"),
			"scanner-cert.pem":    []byte("ScannerCert"),
			"scanner-key.pem":     []byte("ScannerKey"),
			"scanner-db-cert.pem": []byte("ScannerDBCert"),
			"scanner-db-key.pem":  []byte("ScannerDBKey"),
			"jwt-key.pem":         []byte("JWTKey"),
		},
		K8sConfig: &K8sConfig{
			CommonConfig: CommonConfig{
				MainImage:      flavor.MainImage(),
				ScannerImage:   flavor.ScannerImage(),
				ScannerDBImage: flavor.ScannerDBImage(),
			},
			DeploymentFormat: v1.DeploymentFormat_KUBECTL,
		},
	}

	files, err := render(config, scannerOnly, flavor)
	assert.NoError(t, err)

	for _, f := range files {
		assert.Falsef(t, strings.HasPrefix(f.Name, "central/"), "unexpected file %s in scanner only bundle", f.Name)
	}
}

func TestRenderWithDeclarativeConfig(t *testing.T) {
	flavor := testutils.MakeImageFlavorForTest(t)
	config := Config{
		SecretsByteMap: map[string][]byte{
			"ca.pem":              []byte("CA"),
			"ca-key.pem":          []byte("CAKey"),
			"cert.pem":            []byte("CentralCert"),
			"key.pem":             []byte("CentralKey"),
			"central-db-cert.pem": []byte("CentralDBCert"),
			"central-db-key.pem":  []byte("CentralDBKey"),
			"scanner-cert.pem":    []byte("ScannerCert"),
			"scanner-key.pem":     []byte("ScannerKey"),
			"scanner-db-cert.pem": []byte("ScannerDBCert"),
			"scanner-db-key.pem":  []byte("ScannerDBKey"),
			"jwt-key.pem":         []byte("JWTKey"),
		},
		K8sConfig: &K8sConfig{
			CommonConfig: CommonConfig{
				MainImage:      flavor.MainImage(),
				ScannerImage:   flavor.ScannerImage(),
				ScannerDBImage: flavor.ScannerDBImage(),
			},
			DeploymentFormat: v1.DeploymentFormat_KUBECTL,
			DeclarativeConfigMounts: DeclarativeConfigMounts{
				ConfigMaps: []string{"config-map-1", "config-map-2"},
				Secrets:    []string{"secret-1", "secret-2"},
			},
		},
	}

	files, err := render(config, renderAll, flavor)
	assert.NoError(t, err)

	centralFile := filterCentralFile(files)
	require.NotNil(t, centralFile)

	unstructuredObj, err := k8sutil.UnstructuredFromYAML(string(centralFile.Content))
	require.NoError(t, err)
	deployment := &appsv1.Deployment{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), deployment)
	require.NoError(t, err)

	// We currently assume only a single container is part of the central deployment.
	volumeMounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts

	volumes := deployment.Spec.Template.Spec.Volumes

	volumeNames := make(map[string]int, len(volumes))
	for i, volume := range volumes {
		volumeNames[volume.Name] = i
	}
	mountNames := make(map[string]int, len(volumeMounts))
	for i, mount := range volumeMounts {
		mountNames[mount.Name] = i
	}

	for _, cm := range config.K8sConfig.DeclarativeConfigMounts.ConfigMaps {
		assert.Contains(t, volumeNames, cm)
		assert.Contains(t, mountNames, cm)
		assert.NotNil(t, volumes[volumeNames[cm]].ConfigMap)
	}

	for _, secret := range config.K8sConfig.DeclarativeConfigMounts.Secrets {
		assert.Contains(t, volumeNames, secret)
		assert.Contains(t, mountNames, secret)
		assert.NotNil(t, volumes[volumeNames[secret]].Secret)
	}
}

func TestRenderDeclarativeConfigEmpty(t *testing.T) {
	flavor := testutils.MakeImageFlavorForTest(t)
	cases := map[string]Config{
		"empty declarative config mounts": {
			SecretsByteMap: map[string][]byte{
				"ca.pem":              []byte("CA"),
				"ca-key.pem":          []byte("CAKey"),
				"cert.pem":            []byte("CentralCert"),
				"key.pem":             []byte("CentralKey"),
				"central-db-cert.pem": []byte("CentralDBCert"),
				"central-db-key.pem":  []byte("CentralDBKey"),
				"scanner-cert.pem":    []byte("ScannerCert"),
				"scanner-key.pem":     []byte("ScannerKey"),
				"scanner-db-cert.pem": []byte("ScannerDBCert"),
				"scanner-db-key.pem":  []byte("ScannerDBKey"),
				"jwt-key.pem":         []byte("JWTKey"),
			},
			K8sConfig: &K8sConfig{
				CommonConfig: CommonConfig{
					MainImage:      flavor.MainImage(),
					ScannerImage:   flavor.ScannerImage(),
					ScannerDBImage: flavor.ScannerDBImage(),
				},
				DeploymentFormat: v1.DeploymentFormat_KUBECTL,
			},
		},
		"empty literal [] in declarative config mounts": {
			SecretsByteMap: map[string][]byte{
				"ca.pem":              []byte("CA"),
				"ca-key.pem":          []byte("CAKey"),
				"cert.pem":            []byte("CentralCert"),
				"key.pem":             []byte("CentralKey"),
				"central-db-cert.pem": []byte("CentralDBCert"),
				"central-db-key.pem":  []byte("CentralDBKey"),
				"scanner-cert.pem":    []byte("ScannerCert"),
				"scanner-key.pem":     []byte("ScannerKey"),
				"scanner-db-cert.pem": []byte("ScannerDBCert"),
				"scanner-db-key.pem":  []byte("ScannerDBKey"),
				"jwt-key.pem":         []byte("JWTKey"),
			},
			K8sConfig: &K8sConfig{
				CommonConfig: CommonConfig{
					MainImage:      flavor.MainImage(),
					ScannerImage:   flavor.ScannerImage(),
					ScannerDBImage: flavor.ScannerDBImage(),
				},
				DeploymentFormat: v1.DeploymentFormat_KUBECTL,
			},
		},
		"nil array in declarative config mounts": {
			SecretsByteMap: map[string][]byte{
				"ca.pem":              []byte("CA"),
				"ca-key.pem":          []byte("CAKey"),
				"cert.pem":            []byte("CentralCert"),
				"key.pem":             []byte("CentralKey"),
				"central-db-cert.pem": []byte("CentralDBCert"),
				"central-db-key.pem":  []byte("CentralDBKey"),
				"scanner-cert.pem":    []byte("ScannerCert"),
				"scanner-key.pem":     []byte("ScannerKey"),
				"scanner-db-cert.pem": []byte("ScannerDBCert"),
				"scanner-db-key.pem":  []byte("ScannerDBKey"),
				"jwt-key.pem":         []byte("JWTKey"),
			},
			K8sConfig: &K8sConfig{
				CommonConfig: CommonConfig{
					MainImage:      flavor.MainImage(),
					ScannerImage:   flavor.ScannerImage(),
					ScannerDBImage: flavor.ScannerDBImage(),
				},
				DeploymentFormat: v1.DeploymentFormat_KUBECTL,
				DeclarativeConfigMounts: DeclarativeConfigMounts{
					ConfigMaps: nil,
					Secrets:    nil,
				},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			files, err := render(c, renderAll, flavor)
			assert.NoError(t, err)

			centralFile := filterCentralFile(files)
			require.NotNil(t, centralFile)

			deployment := getCentralDeployment(t, centralFile)

			// We currently assume only a single container is part of the central deployment.
			volumeMounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts

			volumes := deployment.Spec.Template.Spec.Volumes

			// Previously, the literal value of "[]" was used as mount name.
			volumeNames := make([]string, 0, len(volumes))
			for _, volume := range volumes {
				volumeNames = append(volumeNames, volume.Name)
			}
			assert.NotContains(t, volumeNames, "[]")

			// No volume mount should exist that starts with the declarative confiugration mount path.
			for _, mount := range volumeMounts {
				assert.False(t, strings.HasPrefix(mount.MountPath, "/run/stackrox.io/declarative-configuration"))
			}
		})
	}
}

func TestDeclarativeConfigDuplicateValues(t *testing.T) {
	flavor := testutils.MakeImageFlavorForTest(t)
	config := Config{
		SecretsByteMap: map[string][]byte{
			"ca.pem":              []byte("CA"),
			"ca-key.pem":          []byte("CAKey"),
			"cert.pem":            []byte("CentralCert"),
			"key.pem":             []byte("CentralKey"),
			"central-db-cert.pem": []byte("CentralDBCert"),
			"central-db-key.pem":  []byte("CentralDBKey"),
			"scanner-cert.pem":    []byte("ScannerCert"),
			"scanner-key.pem":     []byte("ScannerKey"),
			"scanner-db-cert.pem": []byte("ScannerDBCert"),
			"scanner-db-key.pem":  []byte("ScannerDBKey"),
			"jwt-key.pem":         []byte("JWTKey"),
		},
		K8sConfig: &K8sConfig{
			CommonConfig: CommonConfig{
				MainImage:      flavor.MainImage(),
				ScannerImage:   flavor.ScannerImage(),
				ScannerDBImage: flavor.ScannerDBImage(),
			},
			DeploymentFormat: v1.DeploymentFormat_KUBECTL,
			DeclarativeConfigMounts: DeclarativeConfigMounts{
				ConfigMaps: []string{"cm-1", "cm-1", "cm-3", "cm-4"},
				Secrets:    []string{"sec-1", "sec-2", "sec-3", "sec-3"},
			},
		},
	}

	files, err := render(config, renderAll, flavor)
	assert.NoError(t, err)

	centralFile := filterCentralFile(files)
	require.NotNil(t, centralFile)

	deployment := getCentralDeployment(t, centralFile)

	volumes, mounts := getDeclarativeConfigVolumes(deployment)

	expectedVolumes := []string{"cm-1", "cm-3", "cm-4", "sec-1", "sec-2", "sec-3"}

	assert.Len(t, volumes, len(expectedVolumes))
	assert.Len(t, mounts, len(expectedVolumes))
	for _, mount := range mounts {
		assert.Contains(t, expectedVolumes, mount.Name)
	}
	for _, volume := range volumes {
		assert.Contains(t, expectedVolumes, volume.Name)
	}
}

func filterCentralFile(files []*zip.File) *zip.File {
	for _, f := range files {
		if f.Name == "central/01-central-13-deployment.yaml" {
			return f
		}
	}
	return nil
}

func getCentralDeployment(t *testing.T, centralFile *zip.File) *appsv1.Deployment {
	unstructuredObj, err := k8sutil.UnstructuredFromYAML(string(centralFile.Content))
	require.NoError(t, err)
	deployment := &appsv1.Deployment{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), deployment)
	require.NoError(t, err)
	return deployment
}

func getDeclarativeConfigVolumes(deployment *appsv1.Deployment) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := make(map[string]corev1.Volume, len(deployment.Spec.Template.Spec.Volumes))
	for _, v := range deployment.Spec.Template.Spec.Volumes {
		volumes[v.Name] = v
	}
	var declarativeVolumeMounts []corev1.VolumeMount
	var declarativeVolumes []corev1.Volume
	// We currently assume only a single container is part of the central deployment.
	for _, mount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		if strings.HasPrefix(mount.MountPath, "/run/stackrox.io/declarative-configuration") {
			declarativeVolumeMounts = append(declarativeVolumeMounts, mount)
			declarativeVolumes = append(declarativeVolumes, volumes[mount.Name])
		}
	}
	return declarativeVolumes, declarativeVolumeMounts
}
