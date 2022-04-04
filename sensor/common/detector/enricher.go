package detector

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v3"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/sensor/common/detector/metrics"
	"github.com/stackrox/rox/sensor/common/imagecacheutils"
	"github.com/stackrox/rox/sensor/common/scan"
	"google.golang.org/grpc/status"
)

const (
	scanTimeout = 6 * time.Minute
)

type scanResult struct {
	action                 central.ResourceAction
	deployment             *storage.Deployment
	images                 []*storage.Image
	networkPoliciesApplied *augmentedobjs.NetworkPoliciesApplied
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

	return svc.ScanImageInternal(ctx, &v1.ScanImageInternalRequest{
		Image: ci,
	})
}

func scanImageLocal(ctx context.Context, svc v1.ImageServiceClient, ci *storage.ContainerImage) (*v1.ScanImageInternalResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, scanTimeout)
	defer cancel()

	img, err := scan.EnrichLocalImage(ctx, svc, ci)
	return &v1.ScanImageInternalResponse{
		Image: img,
	}, err
}

type scanFunc func(ctx context.Context, svc v1.ImageServiceClient, ci *storage.ContainerImage) (*v1.ScanImageInternalResponse, error)

func scanWithRetries(ctx context.Context, svc v1.ImageServiceClient, ci *storage.ContainerImage, scan scanFunc) (*v1.ScanImageInternalResponse, error) {
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
		scannedImage, err := scan(ctx, svc, ci)
		if err != nil {
			for _, detail := range status.Convert(err).Details() {
				// If the client is effectively rate-limited, backoff and try again.
				if _, isTooManyParallelScans := detail.(*v1.ScanImageInternalResponseDetails_TooManyParallelScans); isTooManyParallelScans {
					time.Sleep(eb.NextBackOff())
					continue outer
				}
			}

			return nil, err
		}

		metrics.ObserveTimeSpentInExponentialBackoff(timeSpentInBackoffSoFar)

		return scannedImage, nil
	}
}

func (c *cacheValue) scanAndSet(ctx context.Context, svc v1.ImageServiceClient, ci *storage.ContainerImage) {
	defer c.signal.Signal()

	// Ask Central to scan the image if the image is not internal.
	// Otherwise, attempt to scan locally.
	scanImageFn := scanImage
	if features.LocalImageScanning.Enabled() && ci.GetIsClusterLocal() {
		scanImageFn = scanImageLocal
	}

	scannedImage, err := scanWithRetries(ctx, svc, ci, scanImageFn)
	if err != nil {
		// Ignore the error and set the image to something basic,
		// so alerting can progress.
		c.image = types.ToImage(ci)
		return
	}

	c.image = scannedImage.GetImage()
}

func newEnricher(cache expiringcache.Cache) *enricher {
	return &enricher{
		scanResultChan: make(chan scanResult),

		imageCache: cache,
		stopSig:    concurrency.NewSignal(),
	}
}

func (e *enricher) getImageFromCache(key string) (*storage.Image, bool) {
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
		// unguarded send (push to channel outside a select) is allowed because the imageChan is a buffered channel of exact size
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

func (e *enricher) blockingScan(deployment *storage.Deployment, netpolApplied *augmentedobjs.NetworkPoliciesApplied, action central.ResourceAction) {
	select {
	case <-e.stopSig.Done():
		return
	case e.scanResultChan <- scanResult{
		action:                 action,
		deployment:             deployment,
		images:                 e.getImages(deployment),
		networkPoliciesApplied: netpolApplied,
	}:
	}
}

func (e *enricher) outputChan() <-chan scanResult {
	return e.scanResultChan
}

func (e *enricher) stop() {
	e.stopSig.Signal()
}
