package heritage

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PastSensor struct {
	ContainerID string    `json:"containerID"`
	PodIP       string    `json:"podIP"`
	Timestamp   time.Time `json:"timestamp"`
}

const (
	cmName             = "sensor-past-instances"
	configMapKey       = "heritage"
	annotationInfoKey  = `stackrox.io/past-sensors-info`
	annotationInfoText = `This data is for sensor to recognize its past pod instances.`
)

var (
	log = logging.LoggerForModule()
)

type Manager struct {
	k8sClient client.Interface
	namespace string

	cachePopulated atomic.Bool
	cache          []PastSensor
}

func NewHeritageManager(ns string, client client.Interface) *Manager {
	m := &Manager{
		cachePopulated: atomic.Bool{},
		k8sClient:      client,
		cache:          make([]PastSensor, 0),
		namespace:      ns,
	}
	return m
}

func (h *Manager) loadCache(ctx context.Context) error {
	if h.cachePopulated.Load() {
		return nil
	}
	data, err := h.read(ctx)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			h.cachePopulated.Store(true)
			log.Debug("No heritage data found. Starting with empty cache")
			return nil
		}
		log.Warnf("Loading data from configMap failed: %v", err)
		h.cachePopulated.Store(false)
		return err
	}
	log.Infof("Sensor heritage data with %d entries loaded to memory", len(data))
	h.cache = append(h.cache, data...)
	h.cachePopulated.Store(true)
	return nil
}

func (h *Manager) GetData(ctx context.Context) []PastSensor {
	if err := h.loadCache(ctx); err != nil {
		log.Warnf("%v", err)
	}
	return h.cache
}

func (h *Manager) Write(ctx context.Context, currentIP, currentContainerID string, now time.Time) error {
	if err := h.loadCache(ctx); err != nil {
		log.Warnf("%v", err)
	}
	data := append(h.cache, PastSensor{
		ContainerID: currentContainerID,
		PodIP:       currentIP,
		Timestamp:   now,
	})
	return h.write(ctx, data...)
}

func (h *Manager) write(ctx context.Context, data ...PastSensor) error {
	cm, err := pastSensorDataToConfigMap(data...)
	if err != nil {
		return errors.Wrapf(err, "converting past sensor data to config map")
	}
	log.Infof("Writing Heritage data %v to ConfigMap %s/%s", data, h.namespace, cmName)
	if errWr := h.writeCM(ctx, cm); errWr != nil {
		if apiErrors.IsNotFound(errWr) {
			log.Infof("Creating ConfigMap %s/%s", h.namespace, cmName)
			if _, errCr := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).
				Create(ctx, cm, metav1.CreateOptions{}); errCr != nil {
				if !apiErrors.IsAlreadyExists(errCr) {
					return errors.Wrapf(errCr, "creating config map %s/%s", h.namespace, cmName)
				}
			}
			return nil
		}
		return errors.Wrapf(errWr, "Failed writing to config map %s/%s", h.namespace, cmName)
	}
	return nil
}

func (h *Manager) writeCM(ctx context.Context, cm *v1.ConfigMap) error {
	if _, err := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).
		Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "updating config map %s/%s", h.namespace, cmName)
	}
	return nil
}

func (h *Manager) read(ctx context.Context) ([]PastSensor, error) {
	cm, err := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return []PastSensor{}, errors.Wrapf(err, "retrieving config map %s/%s", h.namespace, cmName)
	}
	data, err := configMapToPastSensorData(cm)
	if err != nil {
		return []PastSensor{}, errors.Wrapf(err, "converting config map to past sensor data")
	}
	return data, nil
}

func pastSensorDataToConfigMap(data ...PastSensor) (*v1.ConfigMap, error) {
	if data == nil {
		return nil, nil
	}
	dataMap := make(map[string]string, len(data))
	byteEntry, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrapf(err, "marshalling data for %v", data)
	}
	dataMap[configMapKey] = string(byteEntry)

	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cmName,
			Annotations: map[string]string{
				annotationInfoKey: annotationInfoText,
			},
		},
		Data: dataMap,
	}, nil
}

func configMapToPastSensorData(cm *v1.ConfigMap) ([]PastSensor, error) {
	if cm == nil {
		return nil, nil
	}
	data := make([]PastSensor, 0, len(cm.Data))
	for key, jsonStr := range cm.Data {
		if key != configMapKey {
			continue
		}
		var entries []PastSensor
		if err := json.Unmarshal([]byte(jsonStr), &entries); err != nil {
			return nil, errors.Wrapf(err, "unmarshalling data %v", jsonStr)
		}
		data = append(data, PastSensor{})
	}
	return data, nil
}
