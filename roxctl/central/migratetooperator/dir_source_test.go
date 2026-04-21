package migratetooperator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pvcDeploymentYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: central-db
spec:
  template:
    spec:
      volumes:
      - name: disk
        persistentVolumeClaim:
          claimName: my-pvc
`

const hostPathDeploymentYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: central-db
spec:
  template:
    spec:
      nodeSelector:
        kubernetes.io/hostname: worker-1
      volumes:
      - name: disk
        hostPath:
          path: /data/stackrox
`

func TestDirSource_PVCInSubdir(t *testing.T) {
	dir := t.TempDir()
	centralDir := filepath.Join(dir, "central")
	require.NoError(t, os.MkdirAll(centralDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(centralDir, "01-central-12-central-db.yaml"), []byte(pvcDeploymentYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	dep, err := src.CentralDBDeployment()
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
	require.Len(t, dep.Spec.Template.Spec.Volumes, 1)
	require.NotNil(t, dep.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim)
	assert.Equal(t, "my-pvc", dep.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName)
}

func TestDirSource_PVCInRoot(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deployment.yaml"), []byte(pvcDeploymentYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	dep, err := src.CentralDBDeployment()
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
	assert.Equal(t, "my-pvc", dep.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName)
}

func TestDirSource_DeepNesting(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	require.NoError(t, os.MkdirAll(nested, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "db.yaml"), []byte(pvcDeploymentYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	dep, err := src.CentralDBDeployment()
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
}

func TestDirSource_HostPathWithNodeSelector(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "db.yaml"), []byte(hostPathDeploymentYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	dep, err := src.CentralDBDeployment()
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
	assert.Equal(t, map[string]string{"kubernetes.io/hostname": "worker-1"}, dep.Spec.Template.Spec.NodeSelector)
	require.NotNil(t, dep.Spec.Template.Spec.Volumes[0].HostPath)
	assert.Equal(t, "/data/stackrox", dep.Spec.Template.Spec.Volumes[0].HostPath.Path)
}

func TestDirSource_MultiDocYAML(t *testing.T) {
	dir := t.TempDir()
	multiDocYAML := `apiVersion: v1
kind: Secret
metadata:
  name: central-db-password
---
` + pvcDeploymentYAML
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bundle.yaml"), []byte(multiDocYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	dep, err := src.CentralDBDeployment()
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
}

func TestDirSource_NotFound(t *testing.T) {
	dir := t.TempDir()
	otherYAML := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: central
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deployment.yaml"), []byte(otherYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	_, err = src.CentralDBDeployment()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDirSource_YmlExtension(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deployment.yml"), []byte(pvcDeploymentYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	dep, err := src.CentralDBDeployment()
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
}

func TestDirSource_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	src, err := newDirSource(dir)
	require.NoError(t, err)

	_, err = src.CentralDBDeployment()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
