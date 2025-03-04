package derivelocalvalues

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_k8sObjectDescription(t *testing.T) {
	mock := localK8sObjectDescription{
		cache: map[string]map[string]unstructured.Unstructured{
			"kindX": {
				"obj": {
					Object: map[string]interface{}{
						"one": map[string]string{"two": "value"},
					},
				}},
			"kindY": {
				"obj1": {
					Object: map[string]interface{}{
						"number": 42,
					},
				},
			},
			"secret": {
				"s1": {
					Object: map[string]interface{}{
						"data": map[string]any{"test": base64.StdEncoding.EncodeToString([]byte("b64 secret"))},
					},
				},
				"s2": {
					Object: map[string]interface{}{
						"stringData": map[string]any{"test": "plaintext secret"},
					},
				},
			},
		},
	}
	od := newK8sObjectDescription(mock)
	obj := od.evaluate(context.Background(), "kindX", "obj", "{$.one.two}")
	assert.Equal(t, "value", obj)
	obj = od.evaluate(context.Background(), "kindY", "obj1", "{$.number}")
	assert.Equal(t, 42, obj)
	obj = od.evaluate(context.Background(), "kindZ", "obj1", "{$.number}")
	assert.Equal(t, nil, obj)

	assert.Equal(t, []string{"Failed to lookup resource kindZ/obj1: resource type not found"}, od.warnings)

	od.warnings = nil
	obj = od.lookupSecretStringP(context.Background(), "s1", "test")
	require.NotNil(t, obj)
	assert.Equal(t, "b64 secret", *(obj.(*string)))
	obj = od.lookupSecretStringP(context.Background(), "s2", "test")
	require.NotNil(t, obj)
	assert.Equal(t, "plaintext secret", *(obj.(*string)))
	assert.Empty(t, od.warnings)
}
