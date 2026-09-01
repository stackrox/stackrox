package derivelocalvalues

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makeUnstructured(key string, m map[string]any) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]any{key: m}}
}

func Test_derivePublicLocalValuesForCentralServices(t *testing.T) {
	type sm = map[string]any
	mock := localK8sObjectDescription{
		cache: map[string]map[string]unstructured.Unstructured{
			"deployment": {
				"scanner": {Object: sm{"spec": sm{
					"replicas": 42,
					"template": sm{"spec": sm{
						"containers": []any{sm{
							"name":      "scanner",
							"image":     "stackrox/scanner:tag",
							"resources": sm{"cpu": 1, "memory": 2},
						}}}}}}},
				"scanner-db": {Object: sm{"spec": sm{
					"template": sm{"spec": sm{
						"containers": []any{sm{
							"name":      "db",
							"image":     "stackrox/scanner-db:tag",
							"resources": sm{"cpu": 3, "memory": 4},
						}}}}}}},
				"central": {Object: sm{"spec": sm{
					"template": sm{"spec": sm{
						"containers": []any{sm{
							"name": "central",
							"env": []any{
								sm{"name": "ROX_ENABLE_SECURE_METRICS", "value": "true"},
							}}},
						"volumes": []any{sm{
							"configMap": sm{
								"name": "central-config",
							},
							"name": "central-config-volume",
						}},
					}}}}}},
			"hpa": {
				"scanner": makeUnstructured("spec", sm{"minReplicas": 41, "maxReplicas": 43, "other": "nothing"}),
			},
		},
	}

	values, err := derivePublicLocalValuesForCentralServices(context.Background(), "", newK8sObjectDescription(mock))
	require.NoError(t, err)

	t.Run("scanner", func(t *testing.T) {
		// StackRox Scanner (Scanner V2) is retired: no V2 values are derived from the live
		// cluster; it is always emitted as disabled, regardless of any existing V2 deployment.
		scanner := values["scanner"].(sm)
		assert.Equal(t, sm{"disable": true}, scanner)
	})

	t.Run("monitoring", func(t *testing.T) {
		monitoring := values["monitoring"].(sm)
		require.NotNil(t, monitoring)
		openshift := monitoring["openshift"].(sm)
		require.NotNil(t, openshift)
		assert.True(t, openshift["enabled"].(bool))
	})
}
