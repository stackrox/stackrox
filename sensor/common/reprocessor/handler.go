package reprocessor

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/image/cache"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/unimplemented"
)

var (
	log = logging.LoggerForModule()
)

// Handler handles request to reprocess deployment (sent by Central).
//
//go:generate mockgen-wrapper
type Handler interface {
	common.SensorComponent
	ProcessReprocessDeployments(*central.ReprocessDeployment) error
	ProcessInvalidateImageCache(*central.InvalidateImageCache) error
	ProcessRefreshImageCacheTTL(*central.RefreshImageCacheTTL) error
}

// NewHandler returns a new instance of a deployment reprocessor.
func NewHandler(admCtrlSettingsMgr admissioncontroller.SettingsManager, detector detector.Detector, imageCache cache.Image) Handler {
	return &handlerImpl{
		admCtrlSettingsMgr: admCtrlSettingsMgr,
		detector:           detector,
		imageCache:         imageCache,
		stopSig:            concurrency.NewErrorSignal(),
	}
}

type handlerImpl struct {
	unimplemented.Receiver

	admCtrlSettingsMgr admissioncontroller.SettingsManager
	detector           detector.Detector
	imageCache         cache.Image
	stopSig            concurrency.ErrorSignal
}

func (h *handlerImpl) Name() string {
	return "reprocessor.handlerImpl"
}

func (h *handlerImpl) Start() error {
	return nil
}

func (h *handlerImpl) Stop() {
	h.stopSig.Signal()
}

func (h *handlerImpl) Notify(common.SensorComponentEvent) {}

func (h *handlerImpl) Capabilities() []centralsensor.SensorCapability {
	// A new sensor capability to reprocess deployment has not been added. In case of mismatched upgrades,
	// the re-processing is discarded, which is fine.
	return []centralsensor.SensorCapability{
		centralsensor.TargetedImageCacheInvalidation,
	}
}

func (h *handlerImpl) ProcessReprocessDeployments(req *central.ReprocessDeployment) error {
	log.Debug("Received request to reprocess deployments from Central")

	select {
	case <-h.stopSig.Done():
		return errors.Wrap(h.stopSig.Err(), "could not fulfill re-process deployment(s) request")
	default:
		go h.detector.ReprocessDeployments(req.GetDeploymentIds()...)
	}
	return nil
}

func (h *handlerImpl) ProcessInvalidateImageCache(req *central.InvalidateImageCache) error {
	log.Debug("Received request to invalidate image caches")

	select {
	case <-h.stopSig.Done():
		return errors.Wrap(h.stopSig.Err(), "could not fulfill invalidate image cache request")
	default:
		h.admCtrlSettingsMgr.InvalidateImageCache(req.GetImageKeys())

		keysToDelete := make([]cache.Key, 0, len(req.GetImageKeys()))
		for _, image := range req.GetImageKeys() {
			keysToDelete = append(keysToDelete, cacheKeyFromImageKey(image))
		}
		h.imageCache.Remove(keysToDelete...)
	}
	return nil
}

func (h *handlerImpl) ProcessRefreshImageCacheTTL(req *central.RefreshImageCacheTTL) error {
	log.Debug("Received request to refresh image cache TTLs")

	select {
	case <-h.stopSig.Done():
		return errors.Wrap(h.stopSig.Err(), "could not fulfill refresh image cache TTL request")
	default:
		for _, image := range req.GetImageKeys() {
			if key := cacheKeyFromImageKey(image); key != "" {
				h.imageCache.Touch(key)
			}
		}
	}
	return nil
}

// cacheKeyFromImageKey resolves the cache key from an InvalidateImageCache_ImageKey
// proto, applying the V2/V1/fullName precedence based on the FlattenImageData capability.
func cacheKeyFromImageKey(imageKey *central.InvalidateImageCache_ImageKey) cache.Key {
	var key string
	if centralcaps.Has(centralsensor.FlattenImageData) {
		key = imageKey.GetImageIdV2()
	} else {
		key = imageKey.GetImageId()
	}
	if key == "" {
		key = imageKey.GetImageFullName()
	}
	return cache.Key(key)
}

func (h *handlerImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}
