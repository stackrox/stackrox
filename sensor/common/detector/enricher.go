package detector

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/types"
	"golang.org/x/sync/semaphore"
)

const (
	scanTimeout        = 6 * time.Minute
	maxConcurrentScans = 20
)

type scanResult struct {
	action     central.ResourceAction
	deployment *storage.Deployment
	images     []*storage.Image
}

type imageCacheKey struct {
	id, name string
}

type imageChanResult struct {
	image        *storage.Image
	containerIdx int
}

type enricher struct {
	imageSvc       v1.ImageServiceClient
	scanResultChan chan scanResult

	imageCache expiringcache.Cache
	stopSig    concurrency.Signal

	concurrentScanSemaphore *semaphore.Weighted
}

type cacheValue struct {
	signal concurrency.Signal
	image  *storage.Image
}

func (c *cacheValue) waitAndGet() *storage.Image {
	c.signal.Wait()
	return c.image.Clone()
}

func (c *cacheValue) scanAndSet(svc v1.ImageServiceClient, ci *storage.ContainerImage, concurrentScanSemaphore *semaphore.Weighted) {
	defer c.signal.Signal()

	if err := concurrentScanSemaphore.Acquire(concurrency.AsContext(&c.signal), 1); err != nil {
		log.Errorf("error acquiring scan semaphore: %v", err)
		c.image = types.ToImage(ci)
		return
	}
	defer concurrentScanSemaphore.Release(1)
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()
	scannedImage, err := svc.ScanImageInternal(ctx, &v1.ScanImageInternalRequest{
		Image: ci,
	})
	if err != nil {
		c.image = types.ToImage(ci)
		return
	}
	c.image = scannedImage.GetImage()
}

func newEnricher(cache expiringcache.Cache) *enricher {
	return &enricher{
		scanResultChan: make(chan scanResult),

		concurrentScanSemaphore: semaphore.NewWeighted(maxConcurrentScans),
		imageCache:              cache,
		stopSig:                 concurrency.NewSignal(),
	}
}

type cacheKeyProvider interface {
	GetId() string
	GetName() *storage.ImageName
}

func getImageCacheKey(provider cacheKeyProvider) imageCacheKey {
	return imageCacheKey{
		id:   provider.GetId(),
		name: provider.GetName().GetFullName(),
	}
}

func (e *enricher) getImageFromCache(key imageCacheKey) (*storage.Image, bool) {
	value, _ := e.imageCache.Get(key).(*cacheValue)
	if value == nil {
		return nil, false
	}
	return value.waitAndGet(), true
}

func (e *enricher) runScan(containerIdx int, ci *storage.ContainerImage) imageChanResult {
	key := getImageCacheKey(ci)

	// Fast path
	img, ok := e.getImageFromCache(key)
	if ok {
		return imageChanResult{
			image:        img,
			containerIdx: containerIdx,
		}
	}

	newValue := &cacheValue{
		signal: concurrency.NewSignal(),
	}
	value := e.imageCache.GetOrSet(key, newValue).(*cacheValue)
	if newValue == value {
		value.scanAndSet(e.imageSvc, ci, e.concurrentScanSemaphore)
	}
	return imageChanResult{
		image:        value.waitAndGet(),
		containerIdx: containerIdx,
	}
}

func (e *enricher) runImageScanAsync(imageChan chan<- imageChanResult, containerIdx int, ci *storage.ContainerImage) {
	go func() {
		// unguarded send (push to channel outside of a select) is allowed because the imageChan is a buffered channel of exact size
		imageChan <- e.runScan(containerIdx, ci)
	}()
}

func (e *enricher) getImages(deployment *storage.Deployment) []*storage.Image {
	imageChan := make(chan imageChanResult, len(deployment.GetContainers()))
	for idx, container := range deployment.GetContainers() {
		e.runImageScanAsync(imageChan, idx, container.GetImage())
	}
	images := make([]*storage.Image, len(deployment.GetContainers()))
	for i := 0; i < len(deployment.GetContainers()); i++ {
		imgResult := <-imageChan
		image := imgResult.image
		// Overwrite the image name as a workaround to the fact that we fetch the image by ID
		// The ID may actually have many names that refer to it. e.g. busybox:latest and busybox:1.31 could have the
		// exact same id
		image.Name = deployment.Containers[imgResult.containerIdx].GetImage().GetName()
		images[imgResult.containerIdx] = image
	}
	return images
}

func (e *enricher) blockingScan(deployment *storage.Deployment, action central.ResourceAction) {
	select {
	case <-e.stopSig.Done():
		return
	case e.scanResultChan <- scanResult{
		action:     action,
		deployment: deployment,
		images:     e.getImages(deployment),
	}:
	}
}

func (e *enricher) outputChan() <-chan scanResult {
	return e.scanResultChan
}

func (e *enricher) stop() {
	e.stopSig.Signal()
}
