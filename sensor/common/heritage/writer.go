package heritage

import (
	"context"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type cmWriter struct {
	// namespace to write the configmap to
	namespace string
	// k8sClient to use fro writing
	k8sClient corev1.ConfigMapsGetter
}

func (h *cmWriter) Write(ctx context.Context, data ...*SensorMetadata) error {
	cm, err := pastSensorDataToConfigMap(data...)
	if err != nil {
		return errors.Wrap(err, "converting past sensor data to config map")
	}

	if err := h.ensureConfigMapExists(ctx, cm); err != nil {
		return errors.Wrap(err, "preparing configMap for writing")
	}
	if _, err := h.k8sClient.ConfigMaps(h.namespace).
		Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "writing to config map %s/%s", h.namespace, cmName)
	}
	return nil
}

func (h *cmWriter) ensureConfigMapExists(ctx context.Context, cm *v1.ConfigMap) error {
	_, err := h.k8sClient.ConfigMaps(h.namespace).Create(ctx, cm, metav1.CreateOptions{})
	if !apiErrors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "creating config map %s/%s", h.namespace, cmName)
	}
	return nil
}

func (h *cmWriter) Read(ctx context.Context) ([]*SensorMetadata, error) {
	cm, err := h.k8sClient.ConfigMaps(h.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return []*SensorMetadata{}, errors.Wrapf(err, "retrieving config map %s/%s", h.namespace, cmName)
	}
	data, err := configMapToPastSensorData(cm)
	if err != nil {
		return []*SensorMetadata{}, errors.Wrap(err, "converting config map to past sensor data")
	}
	return data, nil
}
