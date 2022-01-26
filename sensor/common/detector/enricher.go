package detector

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v3"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/sensor/common/detector/metrics"
	"github.com/stackrox/rox/sensor/common/imagecacheutils"
	"google.golang.org/grpc/status"
)

const (
	scanTimeout = 6 * time.Minute
)

type scanResult struct {
	action     central.ResourceAction
	deployment *storage.Deployment
	images     []*storage.Image
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
}

type cacheValue struct {
	signal concurrency.Signal
	image  *storage.Image
}

func (c *cacheValue) waitAndGet() *storage.Image {
	c.signal.Wait()
	return c.image
}

func scanImage(ctx context.Context, svc v1.ImageServiceClient, ci *storage.ContainerImage) (*v1.ScanImageInternalResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, scanTimeout)
	defer cancel()

	log.Infof("Scanning image: %v %v %v", ci.GetId(), ci.GetName(), ci.GetNotPullable())
	return svc.ScanImageInternal(ctx, &v1.ScanImageInternalRequest{
		Image: ci,
	})
}

func (c *cacheValue) scanAndSet(ctx context.Context, svc v1.ImageServiceClient, ci *storage.ContainerImage) {
	defer c.signal.Signal()

	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = 5 * time.Second
	eb.Multiplier = 2
	eb.MaxInterval = 4 * time.Minute
	eb.MaxElapsedTime = 0 // Never stop the backoff, leave that decision to the parent context.

	eb.Reset()

outer:
	for {
		// We want to get the time spent in backoff without including the time it took to scan the image.
		timeSpentInBackoffSoFar := eb.GetElapsedTime()
		scannedImage, err := scanImage(ctx, svc, ci)
		if err != nil {
			for _, detail := range status.Convert(err).Details() {
				// If the client is effectively rate limited, backoff and try again.
				if _, isTooManyParallelScans := detail.(*v1.ScanImageInternalResponseDetails_TooManyParallelScans); isTooManyParallelScans {
					time.Sleep(eb.NextBackOff())
					continue outer
				}
			}
			c.image = types.ToImage(ci)
			return
		}
		metrics.ObserveTimeSpentInExponentialBackoff(timeSpentInBackoffSoFar)
		c.image = scannedImage.GetImage()
		return
	}
}

func newEnricher(cache expiringcache.Cache) *enricher {
	return &enricher{
		scanResultChan: make(chan scanResult),

		imageCache: cache,
		stopSig:    concurrency.NewSignal(),
	}
}

func (e *enricher) getImageFromCache(key imagecacheutils.ImageCacheKey) (*storage.Image, bool) {
	value, _ := e.imageCache.Get(key).(*cacheValue)
	if value == nil {
		return nil, false
	}
	return value.waitAndGet(), true
}

func (e *enricher) runScan(containerIdx int, ci *storage.ContainerImage) imageChanResult {
	key := imagecacheutils.GetImageCacheKey(ci)

	// If the container image says that the image is not pullable, don't even bother trying to scan
	if ci.GetNotPullable() {
		return imageChanResult{
			image:        types.ToImage(ci),
			containerIdx: containerIdx,
		}
	}

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
		value.scanAndSet(concurrency.AsContext(&e.stopSig), e.imageSvc, ci)
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

		// This will ensure that when we change the Name of the image
		// that it will not cause a potential race condition
		// cloning the full object is too expensive and also unnecessary
		image := *imgResult.image
		// Overwrite the image Name as a workaround to the fact that we fetch the image by ID
		// The ID may actually have many names that refer to it. e.g. busybox:latest and busybox:1.31 could have the
		// exact same ID
		image.Name = deployment.Containers[imgResult.containerIdx].GetImage().GetName()
		images[imgResult.containerIdx] = &image
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
