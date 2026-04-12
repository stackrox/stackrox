package k8swatch

import (
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
)

// NewInformerAdapterForTest creates an adapter with a custom base URL for testing.
func NewInformerAdapterForTest(baseURL, apiPath string, client *http.Client, newObject func() runtime.Object) *InformerAdapter {
	a := NewInformerAdapter(apiPath, client, newObject)
	a.baseURL = baseURL
	return a
}
