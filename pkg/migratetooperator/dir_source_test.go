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

const centralDeploymentYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: central
spec:
  template:
    spec:
      containers:
      - name: central
        env:
        - name: ROX_ENABLE_SECURE_METRICS
          value: "true"
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

func TestDirSource_DeepNesting(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	require.NoError(t, os.MkdirAll(nested, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "db.yaml"), []byte(pvcDeploymentYAML), 0644))

	src, err := NewDirSource(dir)
	require.NoError(t, err)

	dep, err := src.Deployment("central-db")
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
}

func TestDirSource_HostPathWithNodeSelector(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "db.yaml"), []byte(hostPathDeploymentYAML), 0644))

	src, err := NewDirSource(dir)
	require.NoError(t, err)

	dep, err := src.Deployment("central-db")
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

	src, err := NewDirSource(dir)
	require.NoError(t, err)

	dep, err := src.Deployment("central-db")
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

	src, err := NewDirSource(dir)
	require.NoError(t, err)

	dep, err := src.Deployment("central-db")
	assert.NoError(t, err)
	assert.Nil(t, dep)
}

func TestDirSource_YmlExtension(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deployment.yml"), []byte(pvcDeploymentYAML), 0644))

	src, err := NewDirSource(dir)
	require.NoError(t, err)

	dep, err := src.Deployment("central-db")
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
}

func TestNewDirSource_NonexistentPath(t *testing.T) {
	base := t.TempDir()
	src, err := NewDirSource(filepath.Join(base, "does-not-exist"))
	require.Error(t, err)
	assert.Nil(t, src)
	assert.Contains(t, err.Error(), "accessing directory")
}

func TestNewDirSource_NonDirectoryPath(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-directory")
	require.NoError(t, os.WriteFile(filePath, []byte("not a directory"), 0644))

	src, err := NewDirSource(filePath)
	require.Error(t, err)
	assert.Nil(t, src)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestDirSource_Deployment(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "central.yaml"), []byte(centralDeploymentYAML), 0644))

	src, err := NewDirSource(dir)
	require.NoError(t, err)

	dep, err := src.Deployment("central")
	require.NoError(t, err)
	assert.Equal(t, "central", dep.Name)
}

func TestDirSource_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	src, err := NewDirSource(dir)
	require.NoError(t, err)

	dep, err := src.Deployment("central-db")
	assert.NoError(t, err)
	assert.Nil(t, dep)
}
