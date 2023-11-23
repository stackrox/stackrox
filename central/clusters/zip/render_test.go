package zip

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

var dummyCerts = sensor.Certs{
	Files: map[string][]byte{
		"secrets/ca.pem":                     []byte("ca cert"),
		"secrets/sensor-cert.pem":            []byte("sensor cert"),
		"secrets/sensor-key.pem":             []byte("sensor key"),
		"secrets/collector-cert.pem":         []byte("collector cert"),
		"secrets/collector-key.pem":          []byte("collector key"),
		"secrets/admission-control-cert.pem": []byte("adm ctrl cert"),
		"secrets/admission-control-key.pem":  []byte("adm ctrl key"),
	},
}

func init() {
	testutils.SetExampleVersion(&testing.T{})
}

func TestRenderOpenshiftEnv(t *testing.T) {
	t.Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameDevelopmentBuild)
	t.Run(storage.ClusterType_OPENSHIFT4_CLUSTER.String(), func(t *testing.T) {
		doTestRenderOpenshif(t, storage.ClusterType_OPENSHIFT4_CLUSTER)
	})
}

func getEnvVarValue(vars []coreV1.EnvVar, name string) (string, bool) {
	for _, envVar := range vars {
		if envVar.Name == name {
			return envVar.Value, true
		}
	}
	return "", false
}

func findContainer(containers []coreV1.Container, name string) (cont coreV1.Container, found bool) {
	for _, cont := range containers {
		if cont.Name == name {
			return cont, true
		}
	}
	return coreV1.Container{}, false
}

func doTestRenderOpenshif(t *testing.T, clusterType storage.ClusterType) {
	cluster := &storage.Cluster{
		Name:      "cluster",
		MainImage: "stackrox/main:abc",
		Type:      clusterType,
	}

	baseFiles, err := renderBaseFiles(cluster, clusters.RenderOptions{}, dummyCerts)
	require.NoError(t, err)

	assertOnSensor := func(obj runtime.Object) {
		deployment := obj.(*v1.Deployment)
		sensorCont, foundSensor := findContainer(deployment.Spec.Template.Spec.Containers, "sensor")
		assert.True(t, foundSensor)
		value, exists := getEnvVarValue(sensorCont.Env, env.OpenshiftAPI.EnvVar())
		assert.True(t, exists)
		assert.Equal(t, "true", value)
	}
	assertOnCollector := func(obj runtime.Object) {
		ds := obj.(*v1.DaemonSet)
		complianceCont, foundMain := findContainer(ds.Spec.Template.Spec.Containers, "compliance")
		assert.True(t, foundMain)
		assert.Equal(t, "compliance", complianceCont.Name)

		if clusterType == storage.ClusterType_OPENSHIFT4_CLUSTER {
			nInvCont, found := findContainer(ds.Spec.Template.Spec.Containers, "node-inventory")
			assert.True(t, found, "node-inventory container should exist under collector DS")
			assert.Equal(t, "node-inventory", nInvCont.Name)

			expectedScannerParts := strings.Split(strings.ReplaceAll(complianceCont.Image, "/main:", "/scanner-slim:"), ":")
			assert.Truef(t, strings.HasPrefix(nInvCont.Image, expectedScannerParts[0]), "scanner-slim image (%q) should be from the same registry as main (%q)", nInvCont.Image, complianceCont.Image)

			value, exists := getEnvVarValue(complianceCont.Env, env.NodeInventoryContainerEnabled.EnvVar())
			assert.True(t, exists)
			assert.Equal(t, "true", value, "compliance should have %s=true", env.NodeInventoryContainerEnabled.EnvVar())
		}
	}

	cases := map[string]func(object runtime.Object){
		"sensor.yaml":    assertOnSensor,
		"collector.yaml": assertOnCollector,
	}

	for _, f := range baseFiles {
		assertFunc, ok := cases[f.Name]
		if !ok {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewBuffer(f.Content)))

		yamlBytes, err := reader.Read()
		assert.NoError(t, err)

		obj, _, err := decode(yamlBytes, nil, nil)
		assert.NoError(t, err)
		assertFunc(obj)
	}
}

func TestRenderWithNoCollection(t *testing.T) {
	t.Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameDevelopmentBuild)
	cluster := &storage.Cluster{
		Name:             "cluster",
		MainImage:        "stackrox/main:abc",
		Type:             storage.ClusterType_OPENSHIFT_CLUSTER,
		CollectionMethod: storage.CollectionMethod_NO_COLLECTION,
	}

	baseFiles, err := renderBaseFiles(cluster, clusters.RenderOptions{}, dummyCerts)
	require.NoError(t, err)

	var found bool
	for _, f := range baseFiles {
		if f.Name == "collector-secret.yaml" {
			found = true
			break
		}
	}
	assert.True(t, found)
}
