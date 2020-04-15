package zip

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestRenderOpenshiftEnv(t *testing.T) {
	cluster := &storage.Cluster{
		Name:      "cluster",
		MainImage: "stackrox/main:abc",
		Type:      storage.ClusterType_OPENSHIFT_CLUSTER,
	}

	baseFiles, err := renderBaseFiles(cluster, false, sensor.Certs{})
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
