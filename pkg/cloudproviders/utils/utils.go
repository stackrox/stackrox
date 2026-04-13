package utils

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	nodeGVR = schema.GroupVersionResource{Version: "v1", Resource: "nodes"}
)

// GetAnyNodeLabels returns the labels of an arbitrary node. This is useful
// to extract global labels such as the cluster name.
func GetAnyNodeLabels(ctx context.Context, client dynamic.Interface) (map[string]string, error) {
	nodeList, err := client.Resource(nodeGVR).List(ctx, v1.ListOptions{Limit: 1})
	if err != nil {
		return nil, errors.Wrap(err, "listing nodes")
	}
	if len(nodeList.Items) == 0 {
		return nil, errors.New("no nodes found")
	}

	// Convert unstructured to typed Node
	var node corev1.Node
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(nodeList.Items[0].Object, &node); err != nil {
		return nil, errors.Wrap(err, "converting node to typed object")
	}
	return node.GetLabels(), nil
}
