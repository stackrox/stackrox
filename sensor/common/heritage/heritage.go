package heritage

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	configMapKey       = "heritage"
	annotationInfoKey  = `stackrox.io/past-sensors-info`
	annotationInfoText = `This data is for sensor to recognize its past pod instances.`

	// heritageMinSize is set to 2 as the smallest reasonable minimum covering 1 entry about the past and 1 about the
	// current sensor. Setting this to 1 would disable the historical data and make the heritage feature useless.
	heritageMinSize = 2

	// heritageMaxAge is set to 1 hour to cover for most of the cases when sensor is restarting.
	// Crash-loops with duration of over 1 hour are enough justification for losses of details on the network graph.
	heritageMaxAge = time.Hour
)

var (
	log    = logging.LoggerForModule()
	cmName = env.PastSensorsConfigmapName.Setting()
)

type SensorMetadata struct {
	ContainerID   string    `json:"containerID"`
	PodIP         string    `json:"podIP"`
	SensorStart   time.Time `json:"sensorStart"`
	LatestUpdate  time.Time `json:"latestUpdate"`
	SensorVersion string    `json:"sensorVersion"`
}

// ReverseCompare compares two `SensorMetadata` to use in sorting.
// The resulting order makes more recently updated entries are at the beginning of the slice.
// If there are two entries with the same `LatestUpdate` (can occur only in tests), then other fields define the order.
func (a *SensorMetadata) ReverseCompare(b *SensorMetadata) int {
	return cmp.Or(
		-a.LatestUpdate.Compare(b.LatestUpdate), // more recent update is smaller
		-a.SensorStart.Compare(b.SensorStart),   // more recent start is smaller
		cmp.Compare(a.PodIP, b.PodIP),
		cmp.Compare(a.ContainerID, b.ContainerID),
	)
}

func (a *SensorMetadata) String() string {
	return fmt.Sprintf("(%s, %s) start=%s, lastUpdate=%s",
		a.ContainerID, a.PodIP, a.SensorStart, a.LatestUpdate)
}

func sensorMetadataString(data []*SensorMetadata) string {
	var str strings.Builder
	for i, entry := range data {
		str.WriteString(fmt.Sprintf("[%d]: %s; ", i, entry.String()))
	}
	return str.String()
}

// Using this as one cannot import the client.Interface from 'sensor/kubernetes/client' directly
type k8sClient interface {
	Kubernetes() kubernetes.Interface
}

type Manager struct {
	k8sClient k8sClient
	namespace string

	// Cache the data for the current instance of Sensor
	currentSensor SensorMetadata

	maxSize int
	minSize int
	maxAge  time.Duration

	// Cache the data from the ConfigMap containing all SensorMetadata
	cacheIsPopulated atomic.Bool
	cacheMutex       sync.Mutex
	cache            []*SensorMetadata
}

func NewHeritageManager(ns string, client k8sClient, start time.Time) *Manager {
	maxSize := env.PastSensorsMaxEntries.IntegerSetting()
	if maxSize > 0 && maxSize < heritageMinSize {
		maxSize = heritageMinSize
		log.Warnf("Value of %s is lower than minimum=%d. Applying %d",
			env.PastSensorsMaxEntries.EnvVar(), heritageMinSize, heritageMinSize)
	}
	return &Manager{
		cacheIsPopulated: atomic.Bool{},
		k8sClient:        client,
		cache:            []*SensorMetadata{},
		namespace:        ns,
		currentSensor: SensorMetadata{
			SensorVersion: version.GetMainVersion(),
			SensorStart:   start,
		},
		maxSize: env.PastSensorsMaxEntries.IntegerSetting(),
		minSize: heritageMinSize,
		maxAge:  heritageMaxAge,
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
	log.Infof("Sensor heritage data with %d entries loaded to memory: %s", len(data), sensorMetadataString(data))
	h.cache = append(h.cache, data...)
	h.cacheIsPopulated.Store(true)
	return nil
}

func (h *Manager) GetData(ctx context.Context) []*SensorMetadata {
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
	return h.currentSensor.ContainerID != "" && h.currentSensor.PodIP != ""
}

func (h *Manager) SetCurrentSensorData(currentIP, currentContainerID string) {
	h.currentSensor.PodIP = currentIP
	h.currentSensor.ContainerID = currentContainerID

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
		if entry.ContainerID == h.currentSensor.ContainerID && entry.PodIP == h.currentSensor.PodIP {
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
	h.cacheMutex.Lock()
	defer h.cacheMutex.Unlock()
	if !h.cacheIsPopulated.Load() {
		if err := h.populateCacheFromConfigMap(ctx); err != nil {
			log.Warnf("%v", err)
		}
	}

	if found := h.updateCachedTimestampNoLock(now); !found {
		h.currentSensor.LatestUpdate = now
		h.cache = append(h.cache, &h.currentSensor)
	}
	h.cache = pruneOldHeritageData(h.cache, now, h.maxAge, h.minSize, h.maxSize)

	log.Debugf("Writing Heritage data %s to ConfigMap %s/%s", sensorMetadataString(h.cache), h.namespace, cmName)
	return h.write(ctx, h.cache...)
}

func (h *Manager) write(ctx context.Context, data ...*SensorMetadata) error {
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

func (h *Manager) readConfigMap(ctx context.Context) ([]*SensorMetadata, error) {
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

// pruneOldHeritageData reduces the number of elements in the []*PastSensor slice by removing
// the oldest entries if there are more than `maxSize` and removing entries older than the `maxAge`.
// It does not remove anything until the `minSize` elements are stored.
func pruneOldHeritageData(in []*SensorMetadata, now time.Time, maxAge time.Duration, minSize, maxSize int) []*SensorMetadata {
	if len(in) <= minSize {
		return in
	}
	if maxSize > 0 && minSize > 0 && maxSize < minSize {
		log.Warnf("Heritage cleanup misconfigured: maxSize(%d) < minSize(%d)", maxSize, minSize)
		return in
	}
	if maxSize == 0 && maxAge == 0 {
		return in
	}
	in = slices.SortedFunc[*SensorMetadata](slices.Values(in), func(a *SensorMetadata, b *SensorMetadata) int {
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
func removeOlderThan(in []*SensorMetadata, now time.Time, minSize int, maxAge time.Duration) []*SensorMetadata {
	if !slices.IsSortedFunc(in, func(a *SensorMetadata, b *SensorMetadata) int {
		return a.ReverseCompare(b)
	}) {
		log.Errorf("Programmer error: cannot remove old heritage data for unsorted slice")
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
