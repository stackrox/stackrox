package heritage

import (
	"context"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	"github.com/stackrox/rox/sensor/kubernetes/client"
)

type cmWriter struct {
	// namespace to write the configmap to
	namespace string
	// dynClient to use for writing
	dynClient dynamic.Interface
}

func (h *cmWriter) Write(ctx context.Context, data ...*SensorMetadata) error {
	cm, err := pastSensorDataToConfigMap(data...)
	if err != nil {
		return errors.Wrap(err, "converting past sensor data to config map")
	}

	if err := h.ensureConfigMapExists(ctx, cm); err != nil {
		return errors.Wrap(err, "preparing configMap for writing")
	}

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cm)
	if err != nil {
		return errors.Wrap(err, "converting configmap to unstructured")
	}
	unstructuredCM := &unstructured.Unstructured{Object: unstructuredObj}
	if _, err := h.dynClient.Resource(client.ConfigMapGVR).Namespace(h.namespace).
		Update(ctx, unstructuredCM, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "writing to config map %s/%s", h.namespace, cmName)
	}
	return nil
}

func (h *cmWriter) ensureConfigMapExists(ctx context.Context, cm *v1.ConfigMap) error {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cm)
	if err != nil {
		return errors.Wrap(err, "converting configmap to unstructured")
	}
	unstructuredCM := &unstructured.Unstructured{Object: unstructuredObj}
	_, err = h.dynClient.Resource(client.ConfigMapGVR).Namespace(h.namespace).Create(ctx, unstructuredCM, metav1.CreateOptions{})
	if !apiErrors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "creating config map %s/%s", h.namespace, cmName)
	}
	return nil
}

func (h *cmWriter) Read(ctx context.Context) ([]*SensorMetadata, error) {
	unstructuredCM, err := h.dynClient.Resource(client.ConfigMapGVR).Namespace(h.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return []*SensorMetadata{}, errors.Wrapf(err, "retrieving config map %s/%s", h.namespace, cmName)
	}
	var cm v1.ConfigMap
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCM.Object, &cm); err != nil {
		return []*SensorMetadata{}, errors.Wrap(err, "converting unstructured to configmap")
	}
	data, err := configMapToPastSensorData(&cm)
	if err != nil {
		return []*SensorMetadata{}, errors.Wrap(err, "converting config map to past sensor data")
	}
	return data, nil
}
