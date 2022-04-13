package zip

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stackrox/stackrox/central/clusters"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/image/sensor"
	"github.com/stackrox/stackrox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/stackrox/pkg/env"
	"github.com/stackrox/stackrox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
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
	testbuildinfo.SetForTest(&testing.T{})
	testutils.SetMainVersion(&testing.T{}, "3.0.55.0")
}

func TestRenderOpenshiftEnv(t *testing.T) {
	for _, clusterType := range []storage.ClusterType{storage.ClusterType_OPENSHIFT_CLUSTER, storage.ClusterType_OPENSHIFT4_CLUSTER} {
		t.Run(clusterType.String(), func(t *testing.T) {
			doTestRenderOpenshiftEnv(t, clusterType)
		})
	}
}

func doTestRenderOpenshiftEnv(t *testing.T, clusterType storage.ClusterType) {
	cluster := &storage.Cluster{
		Name:      "cluster",
		MainImage: "stackrox/main:abc",
		Type:      clusterType,
	}

	baseFiles, err := renderBaseFiles(cluster, clusters.RenderOptions{}, dummyCerts)
	require.NoError(t, err)

	for _, f := range baseFiles {
		if f.Name != "sensor.yaml" {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewBuffer(f.Content)))

		yamlBytes, err := reader.Read()
		assert.NoError(t, err)

		obj, _, err := decode(yamlBytes, nil, nil)
		assert.NoError(t, err)

		deployment := obj.(*v1.Deployment)
		var found bool
		for _, envVar := range deployment.Spec.Template.Spec.Containers[0].Env {
			if envVar.Name == env.OpenshiftAPI.EnvVar() {
				found = true
				assert.Equal(t, "true", envVar.Value)
			}
		}
		assert.True(t, found)
	}
}

func TestRenderWithNoCollection(t *testing.T) {
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
