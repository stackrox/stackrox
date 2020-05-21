package manager

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/set"
	admission "k8s.io/api/admission/v1beta1"
)

const (
	imageCacheTTL = 30 * time.Minute
)

type imageCacheEntry struct {
	*storage.Image
	timestamp time.Time
}

func (m *manager) getCachedImage(img *storage.ContainerImage) *storage.Image {
	if img.GetId() == "" {
		return nil
	}

	cachedEntry, ok := m.imageCache.Get(img.GetId())
	if !ok {
		return nil
	}
	cachedImg := cachedEntry.(imageCacheEntry)
	if time.Since(cachedImg.timestamp) > imageCacheTTL {
		m.imageCache.RemoveIf(img.GetId(), func(entry interface{}) bool { return entry == cachedEntry })
		return nil
	}

	return cachedImg.Image
}

func (m *manager) cacheImage(img *storage.Image) {
	if img.GetId() == "" {
		return
	}

	cacheEntry := imageCacheEntry{
		Image:     img,
		timestamp: time.Now(),
	}

	m.imageCache.Add(img.GetId(), cacheEntry)
}

type fetchImageResult struct {
	idx int
	err error
	img *storage.Image
}

func (m *manager) fetchImage(ctx context.Context, s *state, resultChan chan<- fetchImageResult, pendingCount *int32, idx int, image *storage.ContainerImage) {
	defer func() {
		if atomic.AddInt32(pendingCount, -1) == 0 {
			close(resultChan)
		}
	}()

	resp, err := m.client.GetImage(ctx, &sensor.GetImageRequest{
		Image:      image,
		ScanInline: s.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline(),
	})
	if err != nil {
		log.Errorf("error fetching image %q: %v", image.GetName().GetFullName(), err)
		resultChan <- fetchImageResult{
			idx: idx,
			err: err,
		}
		return
	}

	img := resp.GetImage()
	m.cacheImage(img)
	// resultChan is exactly sized so this will be nonblocking
	resultChan <- fetchImageResult{
		idx: idx,
		img: img,
	}
}

func (m *manager) getAvailableImagesAndKickOffScans(ctx context.Context, s *state, deployment *storage.Deployment) ([]*storage.Image, <-chan fetchImageResult) {
	images := make([]*storage.Image, len(deployment.GetContainers()))
	imgChan := make(chan fetchImageResult, len(deployment.GetContainers()))

	pendingCount := int32(1)

	scanInline := s.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline()

	for idx, container := range deployment.GetContainers() {
		image := container.GetImage()
		if image.GetId() != "" || scanInline {
			cachedImage := m.getCachedImage(image)
			if cachedImage != nil {
				images[idx] = cachedImage
			}
			// The cached image might be insufficient if it doesn't have a scan and we want to do inline scans.
			if ctx != nil && (cachedImage == nil || (scanInline && cachedImage.GetScan() == nil)) {
				atomic.AddInt32(&pendingCount, 1)
				go m.fetchImage(ctx, s, imgChan, &pendingCount, idx, image)
			}
		}
		if images[idx] == nil {
			images[idx] = types.ToImage(container.GetImage())
		}
	}

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

	oldK8sObj, err := unmarshalK8sObject(req.Kind, req.OldObject.Raw)
	if err != nil {
		log.Errorf("Failed to unmarshal old object into K8s object: %v", err)
		return true
	}

	oldDeployment, err := resources.NewDeploymentFromStaticResource(oldK8sObj, req.Kind.Kind, s.GetClusterConfig().GetRegistryOverride())
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
