package heritage

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PastSensor struct {
	ContainerID  string    `json:"containerID"`
	PodIP        string    `json:"podIP"`
	SensorStart  time.Time `json:"sensorStart"`
	LatestUpdate time.Time `json:"latestUpdate"`
}

// ReverseCompare sorts so that younger entries are at the beginning of the slice
func (a *PastSensor) ReverseCompare(b *PastSensor) int {
	if n := a.LatestUpdate.Compare(b.LatestUpdate); n != 0 {
		return n * -1
	}
	if n := a.SensorStart.Compare(b.SensorStart); n != 0 {
		return n * -1
	}
	if n := cmp.Compare(a.PodIP, b.PodIP); n != 0 {
		return n
	}
	if n := cmp.Compare(a.ContainerID, b.ContainerID); n != 0 {
		return n
	}
	return 0
}

const (
	cmName             = "sensor-past-instances"
	configMapKey       = "heritage"
	annotationInfoKey  = `stackrox.io/past-sensors-info`
	annotationInfoText = `This data is for sensor to recognize its past pod instances.`

	// TODO: parametrize with env vars?
	heritageMaxSize = 50
	heritageMinSize = 2
	heritageMaxAge  = time.Hour
)

var (
	log = logging.LoggerForModule()
)

type Manager struct {
	k8sClient client.Interface
	namespace string

	mutex sync.Mutex

	// Cache the data for the current instance of Sensor
	currentIP               string
	currentContainerID      string
	lastUpdateOfCurrentData time.Time

	// Cache the data from the ConfigMap about the past instances of Sensor
	cachePopulated atomic.Bool
	cache          []*PastSensor
}

func NewHeritageManager(ns string, client client.Interface) *Manager {
	m := &Manager{
		cachePopulated: atomic.Bool{},
		k8sClient:      client,
		cache:          []*PastSensor{},
		namespace:      ns,
	}
	return m
}

func (h *Manager) loadCacheNoLock(ctx context.Context) error {
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
	log.Infof("Sensor heritage data with %d entries loaded to memory: %v", len(data), data)
	h.cache = append(h.cache, data...)
	h.cachePopulated.Store(true)
	return nil
}

func (h *Manager) GetData(ctx context.Context) []*PastSensor {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if err := h.loadCacheNoLock(ctx); err != nil {
		log.Warnf("%v", err)
	}
	return h.cache
}

func (h *Manager) HasCurrentSensorData() bool {
	return h.currentIP != "" && h.currentContainerID != ""
}

func (h *Manager) SetCurrentSensorData(currentIP, currentContainerID string) {
	h.currentIP = currentIP
	h.currentContainerID = currentContainerID

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := h.UpsertConfigMap(ctx, time.Now()); err != nil {
		log.Warnf("Failed to update heritage data in the configMap: %v", err)
	}
}

// updateCacheNoLock updates the timestamp if container ID and IP already exist in the heritage.
// The size of h.cache is expected to be <10 in most of the cases.
func (h *Manager) updateCacheNoLock(now time.Time) bool {
	for _, entry := range h.cache {
		if entry.ContainerID == h.currentContainerID && entry.PodIP == h.currentIP {
			if entry.SensorStart.IsZero() {
				entry.SensorStart = now
			}
			entry.LatestUpdate = now
			return true
		}
	}
	return false
}

func (h *Manager) UpsertConfigMap(ctx context.Context, now time.Time) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if err := h.loadCacheNoLock(ctx); err != nil {
		log.Warnf("%v", err)
	}

	if !h.updateCacheNoLock(now) {
		h.cache = append(h.cache, &PastSensor{
			ContainerID:  h.currentContainerID,
			PodIP:        h.currentIP,
			SensorStart:  now,
			LatestUpdate: now,
		})
	}
	h.cache = cleanupHeritageData(h.cache, now, heritageMaxAge, heritageMinSize, heritageMaxSize)

	h.lastUpdateOfCurrentData = now
	return h.write(ctx, h.cache...)
}

func (h *Manager) write(ctx context.Context, data ...*PastSensor) error {
	cm, err := pastSensorDataToConfigMap(data...)
	if err != nil {
		return errors.Wrapf(err, "converting past sensor data to config map")
	}
	log.Infof("Writing Heritage data %v to ConfigMap %s/%s", data, h.namespace, cmName)
	if errWr := h.writeConfigMap(ctx, cm); errWr != nil {
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

func (h *Manager) writeConfigMap(ctx context.Context, cm *v1.ConfigMap) error {
	if _, err := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).
		Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "updating config map %s/%s", h.namespace, cmName)
	}
	return nil
}

func (h *Manager) read(ctx context.Context) ([]*PastSensor, error) {
	cm, err := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return []*PastSensor{}, errors.Wrapf(err, "retrieving config map %s/%s", h.namespace, cmName)
	}
	data, err := configMapToPastSensorData(cm)
	if err != nil {
		return []*PastSensor{}, errors.Wrapf(err, "converting config map to past sensor data")
	}
	return data, nil
}

func pastSensorDataToConfigMap(data ...*PastSensor) (*v1.ConfigMap, error) {
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

func configMapToPastSensorData(cm *v1.ConfigMap) ([]*PastSensor, error) {
	if cm == nil {
		return nil, nil
	}
	data := make([]*PastSensor, 0, len(cm.Data))
	for key, jsonStr := range cm.Data {
		if key != configMapKey {
			continue
		}
		var entries []PastSensor
		if err := json.Unmarshal([]byte(jsonStr), &entries); err != nil {
			return nil, errors.Wrapf(err, "unmarshalling data %v", jsonStr)
		}
		for _, entry := range entries {
			data = append(data, &entry)
		}
	}
	return data, nil
}

func cleanupHeritageData(in []*PastSensor, now time.Time, maxAge time.Duration, minSize, maxSize int) []*PastSensor {
	if len(in) <= minSize {
		return in
	}
	if maxSize > 0 && minSize > 0 && maxSize < minSize {
		log.Warnf("Heritage cleanup misconfigured: maxSize < minSize")
		return in
	}
	if maxSize == 0 && maxAge == 0 {
		return in
	}
	in = slices.SortedFunc[*PastSensor](slices.Values(in), func(a *PastSensor, b *PastSensor) int {
		return a.ReverseCompare(b)
	})
	if maxSize > 0 && len(in) > maxSize {
		in = in[:maxSize]
	}
	if maxAge == 0 {
		return in
	}
	return removeOlderThan(in, now, minSize, maxAge)
}
func removeOlderThan(in []*PastSensor, now time.Time, minSize int, maxAge time.Duration) []*PastSensor {
	if !slices.IsSortedFunc(in, func(a *PastSensor, b *PastSensor) int {
		return a.ReverseCompare(b)
	}) {
		log.Errorf("Programmer error: cannot cleanup heritage data for unsorted slice")
		return in
	}
	// The slice is sorted by LastestUpdate
	cutOff := now.Add(-maxAge)
	for idx, entry := range in {
		// We assume that LatestUpdate cannot be zero, as in the worst case we'd have LatestUpdate==SensorStart
		if idx < minSize || entry.LatestUpdate.After(cutOff) {
			continue
		}
		return in[:idx]
	}
	return in
}
