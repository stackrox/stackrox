package heritage

import (
	"context"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Using this as one cannot import the client.Interface from 'sensor/kubernetes/client' directly
type k8sClient interface {
	Kubernetes() kubernetes.Interface
}

type cmWriter struct {
	// namespace to write the configmap to
	namespace string
	// k8sClient to use fro writing
	k8sClient k8sClient
}

func (h *cmWriter) Write(ctx context.Context, data ...*SensorMetadata) error {
	cm, err := pastSensorDataToConfigMap(data...)
	if err != nil {
		return errors.Wrap(err, "converting past sensor data to config map")
	}

	if err := h.ensureConfigMapExists(ctx, cm); err != nil {
		return errors.Wrap(err, "preparing configMap for writing")
	}
	if _, err := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).
		Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "writing to config map %s/%s", h.namespace, cmName)
	}
	return nil
}

func (h *cmWriter) ensureConfigMapExists(ctx context.Context, cm *v1.ConfigMap) error {
	if _, errCr := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).
		Create(ctx, cm, metav1.CreateOptions{}); errCr != nil {
		if !apiErrors.IsAlreadyExists(errCr) {
			return errors.Wrapf(errCr, "creating config map %s/%s", h.namespace, cmName)
		}
	}
	return nil
}

func (h *cmWriter) Read(ctx context.Context) ([]*SensorMetadata, error) {
	cm, err := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return []*SensorMetadata{}, errors.Wrapf(err, "retrieving config map %s/%s", h.namespace, cmName)
	}
	data, err := configMapToPastSensorData(cm)
	if err != nil {
		return []*SensorMetadata{}, errors.Wrap(err, "converting config map to past sensor data")
	}
	return data, nil
}
