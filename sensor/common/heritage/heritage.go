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
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
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
	SensorVersion string    `json:"sensorVersion"`
}

// ReverseCompare compares two `SensorMetadata` to use in sorting.
// The resulting order makes more recently updated entries are at the beginning of the slice.
// If there are two entries with the same `LatestUpdate` (can occur only in tests), then other fields define the order.
func (a *SensorMetadata) ReverseCompare(b *SensorMetadata) int {
	return cmp.Or(
		-a.SensorStart.Compare(b.SensorStart), // more recent start is smaller
		cmp.Compare(a.PodIP, b.PodIP),
		cmp.Compare(a.ContainerID, b.ContainerID),
	)
}

func (a *SensorMetadata) String() string {
	return fmt.Sprintf("(%s, %s) start=%s",
		a.ContainerID, a.PodIP, a.SensorStart)
}

func (a *SensorMetadata) UpdateTimestamps(now time.Time) {
	if a.SensorStart.IsZero() {
		a.SensorStart = now
	}
}

func sensorMetadataString(data []*SensorMetadata) string {
	var str strings.Builder
	for i, entry := range data {
		str.WriteString(fmt.Sprintf("[%d]: %s; ", i, entry.String()))
	}
	return str.String()
}

type configMapWriter interface {
	Write(ctx context.Context, data ...*SensorMetadata) error
	Read(ctx context.Context) ([]*SensorMetadata, error)
}

type Manager struct {
	namespace string

	// Cache the data for the current instance of Sensor
	currentSensor SensorMetadata

	cmWriter configMapWriter

	maxSize int
	minSize int
	maxAge  time.Duration

	// Cache the data from the ConfigMap containing all SensorMetadata
	cacheIsPopulated atomic.Bool
	cacheMutex       sync.Mutex
	cache            []*SensorMetadata
}

func NewHeritageManager(ns string, client corev1.ConfigMapsGetter, start time.Time) *Manager {
	return &Manager{
		cacheIsPopulated: atomic.Bool{},
		cache:            []*SensorMetadata{},
		namespace:        ns,
		currentSensor: SensorMetadata{
			SensorVersion: version.GetMainVersion(),
			SensorStart:   start,
		},
		cmWriter: &cmWriter{
			k8sClient: client,
			namespace: ns,
		},
		maxSize: env.PastSensorsMaxEntries.IntegerSetting(), // Setting value is already validated
		minSize: heritageMinSize,                            // Keep in sync with minimum of `env.PastSensorsMaxEntries`
		maxAge:  heritageMaxAge,
	}
}

func (h *Manager) populateCacheFromConfigMapNoLock(ctx context.Context) error {
	data, err := h.cmWriter.Read(ctx)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			h.cacheIsPopulated.Store(true)
			log.Debug("No heritage data found. Starting with empty cache.")
			return nil
		}
		log.Warnf("Loading data from configMap failed: %v", err)
		h.cacheIsPopulated.Store(false)
		return errors.Wrap(err, "reading configmap")
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
	if err := h.populateCacheFromConfigMapNoLock(ctx); err != nil {
		log.Warnf("%v", err)
	}
	return h.cache
}

func (h *Manager) SetCurrentSensorData(currentIP, currentContainerID string) {
	h.currentSensor.PodIP = currentIP
	h.currentSensor.ContainerID = currentContainerID
	if h.maxSize == 0 {
		return // feature disabled
	}
	now := time.Now()
	h.currentSensor.UpdateTimestamps(now)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := h.upsertConfigMap(ctx, now); err != nil {
		log.Warnf("Failed to update heritage data in the configMap: %v", err)
	}
}

// getCurrentSensorIndexInCache checks whether an entry with given containerID and podIP exists in cache and returns its slice index.
func (h *Manager) getCurrentSensorIndexInCache() (found bool, index int) {
	for idx, entry := range h.cache {
		if entry.ContainerID == h.currentSensor.ContainerID && entry.PodIP == h.currentSensor.PodIP {
			return true, idx
		}
	}
	return false, 0
}

func (h *Manager) upsertConfigMap(ctx context.Context, now time.Time) (wrote bool, err error) {
	h.cacheMutex.Lock()
	defer h.cacheMutex.Unlock()
	if !h.cacheIsPopulated.Load() {
		if err := h.populateCacheFromConfigMapNoLock(ctx); err != nil {
			log.Warnf("%v", err)
		}
	}
	if updated := h.updateCache(now); !updated {
		return false, nil
	}
	err = h.write(ctx)
	if err == nil {
		log.Debugf("Wrote heritage data %s to ConfigMap %s/%s", sensorMetadataString(h.cache), h.namespace, cmName)
	}
	return true, err
}

// updateCache adds the currentSensor data to cache, prunes old data (according to settings) and returns true.
// If the cache already contains data of currentSensor and there are no updates to its containerID or podIP,
// then the configMap is not written and false is returned.
func (h *Manager) updateCache(now time.Time) bool {
	cacheHit, idx := h.getCurrentSensorIndexInCache()
	if cacheHit {
		h.cache[idx].UpdateTimestamps(now)
		// Limit requests to k8s API and do not update configMap if only the timestamps have changed.
		return false
	}

	// Current sensor is not found in cache or there is an update to containerID or podIP.
	h.cache = append(h.cache, &h.currentSensor)
	h.cache = pruneOldHeritageData(h.cache, now, h.maxAge, h.minSize, h.maxSize)
	return true
}

// write writes the cache contents into configMap.
func (h *Manager) write(ctx context.Context) error {
	if err := h.cmWriter.Write(ctx, h.cache...); err != nil {
		return errors.Wrap(err, "writing configmap")
	}
	return nil
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
	// The slice is sorted by `SensorStart`
	cutOff := now.Add(-maxAge)
	for idx, entry := range in {
		// We assume that SensorStart cannot be zero
		if idx < minSize || entry.SensorStart.After(cutOff) || entry.SensorStart.Equal(cutOff) {
			continue
		}
		return in[:idx]
	}
	return in
}
