package heritage

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PastSensor struct {
	ContainerID   string    `json:"containerID"`
	PodIP         string    `json:"podIP"`
	SensorStart   time.Time `json:"sensorStart"`
	LatestUpdate  time.Time `json:"latestUpdate"`
	SensorVersion string    `json:"sensorVersion"`
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
	heritageMaxSize = 20
	heritageMinSize = 2
	heritageMaxAge  = time.Hour
)

var (
	log = logging.LoggerForModule()
)

// Using this as one cannot import the client.Interface from 'sensor/kubernetes/client' directly
type k8sClient interface {
	Kubernetes() kubernetes.Interface
}

type Manager struct {
	k8sClient k8sClient
	namespace string

	// Cache the data for the current instance of Sensor
	currentIP               string
	currentContainerID      string
	sensorStart             time.Time
	sensorVersion           string
	lastUpdateOfCurrentData time.Time

	// Cache the data from the ConfigMap about the past instances of Sensor
	cacheIsPopulated atomic.Bool
	cacheMutex       sync.Mutex
	cache            []*PastSensor
}

func NewHeritageManager(ns string, client k8sClient, start time.Time) *Manager {
	return &Manager{
		cacheIsPopulated: atomic.Bool{},
		k8sClient:        client,
		cache:            []*PastSensor{},
		namespace:        ns,
		sensorStart:      start,
		sensorVersion:    version.GetMainVersion(),
	}
}

func (h *Manager) populateCacheFromConfigMap(ctx context.Context) error {
	data, err := h.readConfigMap(ctx)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			h.cacheIsPopulated.Store(true)
			log.Debug("No heritage data found. Starting with empty cache")
			return nil
		}
		log.Warnf("Loading data from configMap failed: %v", err)
		h.cacheIsPopulated.Store(false)
		return err
	}
	log.Infof("Sensor heritage data with %d entries loaded to memory: %s", len(data), pastSensorDataString(data))
	h.cache = append(h.cache, data...)
	h.cacheIsPopulated.Store(true)
	return nil
}

func (h *Manager) GetData(ctx context.Context) []*PastSensor {
	h.cacheMutex.Lock()
	defer h.cacheMutex.Unlock()
	if h.cacheIsPopulated.Load() {
		return h.cache
	}
	if err := h.populateCacheFromConfigMap(ctx); err != nil {
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

// updateCachedTimestampNoLock updates the timestamp if container ID and IP already exist in the heritage.
// The size of h.cache is expected to be <10 in most of the cases.
func (h *Manager) updateCachedTimestampNoLock(now time.Time) bool {
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
	h.cacheMutex.Lock()
	defer h.cacheMutex.Unlock()
	if !h.cacheIsPopulated.Load() {
		if err := h.populateCacheFromConfigMap(ctx); err != nil {
			log.Warnf("%v", err)
		}
	}

	if found := h.updateCachedTimestampNoLock(now); !found {
		h.cache = append(h.cache, &PastSensor{
			ContainerID:   h.currentContainerID,
			PodIP:         h.currentIP,
			SensorStart:   h.sensorStart,
			SensorVersion: h.sensorVersion,
			LatestUpdate:  now,
		})
	}
	h.cache = cleanupHeritageData(h.cache, now, heritageMaxAge, heritageMinSize, heritageMaxSize)

	h.lastUpdateOfCurrentData = now
	log.Debugf("Writing Heritage data %s to ConfigMap %s/%s", pastSensorDataString(h.cache), h.namespace, cmName)
	return h.write(ctx, h.cache...)
}

func pastSensorDataString(data []*PastSensor) string {
	str := ""
	for i, entry := range data {
		str = fmt.Sprintf("%s[%d]: (%s, %s) start=%s, lastUpdate=%s; ",
			str, i, entry.ContainerID, entry.PodIP, entry.SensorStart, entry.LatestUpdate)
	}
	return str
}

func (h *Manager) write(ctx context.Context, data ...*PastSensor) error {
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

func (h *Manager) ensureConfigMapExists(ctx context.Context, cm *v1.ConfigMap) error {
	if _, errCr := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).
		Create(ctx, cm, metav1.CreateOptions{}); errCr != nil {
		if !apiErrors.IsAlreadyExists(errCr) {
			return errors.Wrapf(errCr, "creating config map %s/%s", h.namespace, cmName)
		}
	}
	return nil
}

func (h *Manager) readConfigMap(ctx context.Context) ([]*PastSensor, error) {
	cm, err := h.k8sClient.Kubernetes().CoreV1().ConfigMaps(h.namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return []*PastSensor{}, errors.Wrapf(err, "retrieving config map %s/%s", h.namespace, cmName)
	}
	data, err := configMapToPastSensorData(cm)
	if err != nil {
		return []*PastSensor{}, errors.Wrap(err, "converting config map to past sensor data")
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

// cleanupHeritageData reduces the number of elements in the []*PastSensor slice by removing
// the oldest entries if there are more than `maxSize` and removing entries older than the `maxAge`.
// It does not remove anything until the `minSize` elements are stored.
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
