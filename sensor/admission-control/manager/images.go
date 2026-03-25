package manager

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc/connectivity"
	admission "k8s.io/api/admission/v1"
)

const (
	imageCacheTTL = 30 * time.Minute
)

type imageCacheEntry struct {
	*storage.Image
	timestamp time.Time
}

// getCachedImage looks up a previously enriched image in the two-level cache.
//
// Resolution order:
//  1. Digest-based refs (Id non-empty): derive the imageCache key directly from the digest
//     (or a V2 UUID5 when FlattenImageData is enabled).
//  2. Tag-only refs (Id empty, e.g. "nginx:1.25"): consult the imageNameToImageCacheKey
//     LRU, which maps full image names to their resolved imageCache keys. This map is
//     populated by cacheImage after enrichment and avoids redundant fetches for the same
//     tag across reviews.
//  3. Tag-only refs with the name cache disabled: skip (no way to resolve without a digest).
//
// On imageCache miss or TTL expiry for a tag-only lookup, the stale name→key mapping is
// removed so the next request triggers a fresh fetch.
//
// The observe flag controls whether cache metrics are emitted. It is false when called
// from within the Coalescer callback (fetchImage) to avoid double-counting, since the
// outer call in getAvailableImagesAndKickOffScans already records the metric.
func (m *manager) getCachedImage(img *storage.ContainerImage, s *state, observe bool) *storage.Image {
	emit := func(fn func()) {
		if observe {
			fn()
		}
	}

	var id string
	if img.GetId() != "" {
		id = img.GetId()
		if s.GetFlattenImageData() {
			id = utils.NewImageV2ID(img.GetName(), img.GetId())
		}
	} else if m.imageNameCacheEnabled {
		cacheKey, ok := m.imageNameToImageCacheKey.Get(img.GetName().GetFullName())
		if !ok {
			emit(observeCacheSkip)
			return nil
		}
		id = cacheKey
	} else {
		emit(observeCacheSkip)
		return nil
	}

	cachedImg, ok := m.imageCache.Get(id)
	if !ok {
		// imageCache entry was LRU-evicted. Clean up the name→key mapping only for
		// tag-only refs (Id empty), since those are the only lookups that went through
		// imageNameToImageCacheKey. Digest-based refs bypass the name map entirely.
		if img.GetId() == "" {
			m.imageNameToImageCacheKey.Remove(img.GetName().GetFullName())
		}
		emit(observeCacheMiss)
		return nil
	}
	if time.Since(cachedImg.timestamp) > imageCacheTTL {
		m.imageCache.RemoveIf(id, func(entry imageCacheEntry) bool { return entry == cachedImg })
		// imageCache entry TTL-expired. Same reasoning as above: only tag-only refs
		// have a name→key mapping to invalidate.
		if img.GetId() == "" {
			m.imageNameToImageCacheKey.Remove(img.GetName().GetFullName())
		}
		emit(observeCacheExpired)
		return nil
	}

	emit(observeCacheHit)
	return cachedImg.Image
}

func (m *manager) cacheImage(scannedImg *storage.Image, containerImageFullName string, s *state) {
	// For tag-only images Central's enricher populates Metadata.V2.Digest but
	// does not set Image.Id. Fall back to the metadata digest so we can still
	// cache enriched results.
	id := utils.GetSHA(scannedImg)
	if id == "" {
		return
	}

	if s.GetFlattenImageData() {
		id = utils.NewImageV2ID(scannedImg.GetName(), id)
	}

	m.imageCache.Add(id, imageCacheEntry{
		Image:     scannedImg,
		timestamp: time.Now(),
	})

	if m.imageNameCacheEnabled && containerImageFullName != "" {
		m.imageNameToImageCacheKey.Add(containerImageFullName, id)
	}
}

type fetchImageResult struct {
	idx int
	err error
	img *storage.Image
}

func (m *manager) getImageFromSensorOrCentral(ctx context.Context, s *state, img *storage.ContainerImage, deployment *storage.Deployment) (*storage.Image, error) {
	// Talk to central if we know its endpoint (and the client connection is not shutting down), and if we are not
	// currently connected to sensor.
	// Note: Sensor is required to scan images in the local registry.
	if !m.sensorConnStatus.Get() && s.centralConn != nil && s.centralConn.GetState() != connectivity.Shutdown {
		start := time.Now()
		resp, err := v1.NewImageServiceClient(s.centralConn).ScanImageInternal(ctx, &v1.ScanImageInternalRequest{
			Image:      img,
			CachedOnly: !s.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline(),
		})
		observeImageFetch(fetchSourceCentral, time.Since(start), err)
		if err != nil {
			return nil, errors.Wrap(err, "scanning image via central")
		}
		return resp.GetImage(), nil
	}

	start := time.Now()
	resp, err := m.client.GetImage(ctx, &sensor.GetImageRequest{
		Image:      img,
		ScanInline: s.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline(),
		Namespace:  deployment.GetNamespace(),
	})
	observeImageFetch(fetchSourceSensor, time.Since(start), err)
	if err != nil {
		return nil, errors.Wrap(err, "getting image from sensor")
	}
	return resp.GetImage(), nil
}

// imageKey returns the key used for coalescing and cache lookup.
//   - Tag-only refs (Id empty): returns the full image name (e.g. "docker.io/library/nginx:1.25").
//     After enrichment, cacheImage maps this name to the resolved digest via imageNameToImageCacheKey.
//   - Digest refs with FlattenImageData: returns a V2 UUID5 derived from name + digest.
//   - Digest refs without FlattenImageData: returns the raw digest.
func (m *manager) imageKey(img *storage.ContainerImage, s *state) string {
	id := img.GetId()

	if id == "" {
		return img.GetName().GetFullName()
	}

	if s.GetFlattenImageData() {
		return utils.NewImageV2ID(img.GetName(), id)
	}

	return id
}

func (m *manager) fetchImage(ctx context.Context, s *state, resultChan chan<- fetchImageResult, pendingCount *int32, idx int, image *storage.ContainerImage, deployment *storage.Deployment) {
	defer func() {
		if atomic.AddInt32(pendingCount, -1) == 0 {
			close(resultChan)
		}
	}()

	imgKey := m.imageKey(image, s)
	scannedImg, err := m.imageFetchGroup.Coalesce(ctx, imgKey, func() (*storage.Image, error) {
		if cached := m.getCachedImage(image, s, false); cached != nil {
			return cached, nil
		}
		img, err := m.getImageFromSensorOrCentral(ctx, s, image, deployment)
		if err != nil {
			return nil, err
		}
		// Caching inside the Coalesce callback ensures only the leader goroutine
		// writes to imageCache. Waiters receive the result from the coalescer,
		// avoiding N-1 redundant cache writes under concurrent bursts.
		m.cacheImage(img, image.GetName().GetFullName(), s)
		return img, nil
	})

	if err != nil {
		log.Errorf("error fetching image %q: %v", image.GetName().GetFullName(), err)
		resultChan <- fetchImageResult{
			idx: idx,
			err: err,
		}
		return
	}

	// resultChan is exactly sized so this will be nonblocking
	resultChan <- fetchImageResult{
		idx: idx,
		img: scannedImg,
	}
}

func (m *manager) getAvailableImagesAndKickOffScans(ctx context.Context, shouldFetch bool, s *state, deployment *storage.Deployment) ([]*storage.Image, <-chan fetchImageResult) {
	images := make([]*storage.Image, len(deployment.GetContainers()))
	imgChan := make(chan fetchImageResult, len(deployment.GetContainers()))

	pendingCount := int32(1)
	fetchCount := 0

	scanInline := s.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline()

	for idx, container := range deployment.GetContainers() {
		image := container.GetImage()
		if image.GetId() != "" || scanInline {
			cachedImage := m.getCachedImage(image, s, true)
			if cachedImage != nil {
				images[idx] = cachedImage
			}
			if shouldFetch && (cachedImage == nil || (scanInline && cachedImage.GetScan() == nil)) {
				atomic.AddInt32(&pendingCount, 1)
				fetchCount++
				go m.fetchImage(ctx, s, imgChan, &pendingCount, idx, image, deployment)
			}
		}
		if images[idx] == nil {
			images[idx] = types.ToImage(container.GetImage())
		}
	}

	observeImageFetchesPerReview(fetchCount)

	if atomic.AddInt32(&pendingCount, -1) == 0 {
		close(imgChan)
	}
	return images, imgChan
}

// hasModifiedImages checks if the given deployment has any new images that the old version was not previously using.
// If there is no old deployment version, or some error is encountered during conversion, true is conservatively
// returned.
func hasModifiedImages(s *state, deployment *storage.Deployment, req *admission.AdmissionRequest) bool {
	if req.OldObject.Raw == nil {
		return true
	}

	if req.SubResource != "" && req.SubResource == ScaleSubResource {
		// TODO: We could consider returning false here since when the admission review request is for the scale
		// subresource, I do not believe it is possible for a user to change the image on the deployment at the same
		// time as updating the scale subresource However, the contract of this function as designed was to be
		// conservative and return true.
		return true
	}

	oldK8sObj, err := unmarshalK8sObject(req.Kind, req.OldObject.Raw)
	if err != nil {
		log.Errorf("Failed to unmarshal old object into K8s object: %v", err)
		return true
	}

	oldDeployment, err := resources.NewDeploymentFromStaticResource(oldK8sObj, req.Kind.Kind, s.clusterID(), s.GetClusterConfig().GetRegistryOverride())
	if err != nil {
		log.Errorf("Failed to convert old K8s object into StackRox deployment: %v", err)
		return true
	}
	if oldDeployment == nil {
		return true
	}

	oldImages := set.NewStringSet()
	for _, container := range oldDeployment.GetContainers() {
		oldImages.Add(container.GetImage().GetName().GetFullName())
	}

	for _, container := range deployment.GetContainers() {
		if !oldImages.Contains(container.GetImage().GetName().GetFullName()) {
			return true
		}
	}

	return false
}

func (m *manager) kickOffImgScansAndDetect(
	ctx context.Context,
	shouldFetch bool,
	s *state,
	getAlertsFunc func(*storage.Deployment, []*storage.Image) ([]*storage.Alert, error),
	deployment *storage.Deployment,
) ([]*storage.Alert, error) {
	if deployment == nil {
		return nil, nil
	}
	images, resultChan := m.getAvailableImagesAndKickOffScans(ctx, shouldFetch, s, deployment)
	alerts, err := getAlertsFunc(deployment, images)

	if !shouldFetch {
		return filterOutUnenrichedImageAlerts(alerts), err
	}

resultsLoop:
	// The results loop continues while this returns true, waiting for enrichment data to resolve them.
	for hasOnlyUnenrichedImageAlerts(alerts) && err == nil {
		select {
		case nextRes, ok := <-resultChan:
			if !ok {
				break resultsLoop
			}
			if nextRes.err != nil {
				continue
			}
			images[nextRes.idx] = nextRes.img

		case <-ctx.Done():
			break resultsLoop
		}

		alerts, err = getAlertsFunc(deployment, images)
	}
	return alerts, err
}
