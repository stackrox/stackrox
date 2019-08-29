package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	serviceAccountFromBundleYAML = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sensor
  namespace: stackrox
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: "sensor"
imagePullSecrets:
- name: stackrox
`

	modifiedServiceAccountFromBundleYAML = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sensor
  namespace: stackroxxx
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: "sensor"
imagePullSecrets:
- name: stackrox
`

	liveServiceAccountYAML = `
apiVersion: v1
imagePullSecrets:
- name: stackrox
kind: ServiceAccount
metadata:
  creationTimestamp: "2019-08-29T08:45:47Z"
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: sensor
  name: sensor
  namespace: stackrox
  resourceVersion: "444536"
  selfLink: /api/v1/namespaces/stackrox/serviceaccounts/sensor
  uid: 6543d0c6-ca39-11e9-a14d-025000000001
secrets:
- name: sensor-token-dsvb7
`
)

func fromYAML(t *testing.T, yamlStr string) *unstructured.Unstructured {
	jsonBytes, err := yaml.ToJSON([]byte(yamlStr))
	require.NoError(t, err)

	obj, _, err := unstructured.UnstructuredJSONScheme.Decode(jsonBytes, nil, nil)
	require.NoError(t, err)
	return obj.(*unstructured.Unstructured)
}

func TestNormalizeObjects_EqualAfterNormalize(t *testing.T) {
	t.Parallel()

	liveSA := fromYAML(t, liveServiceAccountYAML)
	saFromBundle := fromYAML(t, serviceAccountFromBundleYAML)

	normalizeObject(liveSA)
	normalizeObject(saFromBundle)
	assert.Equal(t, liveSA, saFromBundle)
}

func TestNormalizeObjects_NotEqualAfterNormalize(t *testing.T) {
	t.Parallel()

	liveSA := fromYAML(t, liveServiceAccountYAML)
	modifiedSAFromBundle := fromYAML(t, modifiedServiceAccountFromBundleYAML)

	normalizeObject(liveSA)
	normalizeObject(modifiedSAFromBundle)
	assert.NotEqual(t, liveSA, modifiedSAFromBundle)
}
