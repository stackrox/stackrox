package migratetooperator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirSource_PVC(t *testing.T) {
	dir := t.TempDir()
	centralDir := filepath.Join(dir, "central")
	require.NoError(t, os.MkdirAll(centralDir, 0755))

	deploymentYAML := `apiVersion: apps/v1
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
	require.NoError(t, os.WriteFile(filepath.Join(centralDir, "01-central-12-central-db.yaml"), []byte(deploymentYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	dep, err := src.CentralDBDeployment()
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)

	require.Len(t, dep.Spec.Template.Spec.Volumes, 1)
	require.NotNil(t, dep.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim)
	assert.Equal(t, "my-pvc", dep.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName)
}

func TestDirSource_HostPathWithNodeSelector(t *testing.T) {
	dir := t.TempDir()
	centralDir := filepath.Join(dir, "central")
	require.NoError(t, os.MkdirAll(centralDir, 0755))

	deploymentYAML := `apiVersion: apps/v1
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
	require.NoError(t, os.WriteFile(filepath.Join(centralDir, "01-central-12-central-db.yaml"), []byte(deploymentYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	dep, err := src.CentralDBDeployment()
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
	assert.Equal(t, map[string]string{"kubernetes.io/hostname": "worker-1"}, dep.Spec.Template.Spec.NodeSelector)

	require.Len(t, dep.Spec.Template.Spec.Volumes, 1)
	require.NotNil(t, dep.Spec.Template.Spec.Volumes[0].HostPath)
	assert.Equal(t, "/data/stackrox", dep.Spec.Template.Spec.Volumes[0].HostPath.Path)
}

func TestDirSource_MultiDocYAML(t *testing.T) {
	dir := t.TempDir()
	centralDir := filepath.Join(dir, "central")
	require.NoError(t, os.MkdirAll(centralDir, 0755))

	multiDocYAML := `apiVersion: v1
kind: Secret
metadata:
  name: central-db-password
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: central-db
spec:
  template:
    spec:
      volumes:
      - name: disk
        persistentVolumeClaim:
          claimName: central-db
`
	require.NoError(t, os.WriteFile(filepath.Join(centralDir, "01-central-12-central-db.yaml"), []byte(multiDocYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	dep, err := src.CentralDBDeployment()
	require.NoError(t, err)
	assert.Equal(t, "central-db", dep.Name)
}

func TestDirSource_NotFound(t *testing.T) {
	dir := t.TempDir()
	centralDir := filepath.Join(dir, "central")
	require.NoError(t, os.MkdirAll(centralDir, 0755))

	otherYAML := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: central
`
	require.NoError(t, os.WriteFile(filepath.Join(centralDir, "deployment.yaml"), []byte(otherYAML), 0644))

	src, err := newDirSource(dir)
	require.NoError(t, err)

	_, err = src.CentralDBDeployment()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
