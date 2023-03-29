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
	for _, clusterType := range []storage.ClusterType{storage.ClusterType_OPENSHIFT_CLUSTER, storage.ClusterType_OPENSHIFT4_CLUSTER} {
		t.Run(clusterType.String(), func(t *testing.T) {
			doTestRenderOpenshif(t, clusterType)
		})
	}
}

func getEnvVarValue(vars []coreV1.EnvVar, name string) (string, bool) {
	for _, envVar := range vars {
		if envVar.Name == name {
			return envVar.Value, true
		}
	}
	return "", false
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
		value, exists := getEnvVarValue(deployment.Spec.Template.Spec.Containers[0].Env, env.OpenshiftAPI.EnvVar())
		assert.True(t, exists)
		assert.Equal(t, "true", value)
	}
	assertOnCollector := func(obj runtime.Object) {
		ds := obj.(*v1.DaemonSet)
		if clusterType == storage.ClusterType_OPENSHIFT4_CLUSTER {
			assert.Len(t, ds.Spec.Template.Spec.Containers, 3)
			mainImage := ds.Spec.Template.Spec.Containers[1].Image
			nodeInvCont := ds.Spec.Template.Spec.Containers[2]
			expectedScannerParts := strings.Split(strings.ReplaceAll(mainImage, "/main:", "/scanner-slim:"), ":")
			assert.Truef(t, strings.HasPrefix(nodeInvCont.Image, expectedScannerParts[0]), "scanner-slim image (%q) should be from the same registry as main (%q)", nodeInvCont.Image, mainImage)
		} else {
			assert.Len(t, ds.Spec.Template.Spec.Containers, 2)
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
