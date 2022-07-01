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
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/detector/metrics"
	"github.com/stackrox/rox/sensor/common/imagecacheutils"
	"github.com/stackrox/rox/sensor/common/scan"
	"github.com/stackrox/rox/sensor/common/store"
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

	serviceAccountStore store.ServiceAccountStore
	imageCache          expiringcache.Cache
	stopSig             concurrency.Signal
}

type cacheValue struct {
	signal concurrency.Signal
	image  *storage.Image
}

func (c *cacheValue) waitAndGet() *storage.Image {
	c.signal.Wait()
	return c.image
}

func scanImage(ctx context.Context, svc v1.ImageServiceClient, req *scanImageRequest) (*v1.ScanImageInternalResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, scanTimeout)
	defer cancel()

	return svc.ScanImageInternal(ctx, &v1.ScanImageInternalRequest{
		Image:            req.containerImage,
		ClusterId:        req.clusterID,
		Namespace:        req.namespace,
		ImagePullSecrets: req.pullSecrets,
	})
}

func scanImageLocal(ctx context.Context, svc v1.ImageServiceClient, req *scanImageRequest) (*v1.ScanImageInternalResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, scanTimeout)
	defer cancel()

	img, err := scan.EnrichLocalImage(ctx, svc, req.containerImage)
	return &v1.ScanImageInternalResponse{
		Image: img,
	}, err
}

type scanFunc func(ctx context.Context, svc v1.ImageServiceClient, req *scanImageRequest) (*v1.ScanImageInternalResponse, error)

func scanWithRetries(ctx context.Context, svc v1.ImageServiceClient, req *scanImageRequest, scan scanFunc) (*v1.ScanImageInternalResponse, error) {
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
		scannedImage, err := scan(ctx, svc, req)
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

func (c *cacheValue) scanAndSet(ctx context.Context, svc v1.ImageServiceClient, req *scanImageRequest) {
	defer c.signal.Signal()

	// Ask Central to scan the image if the image is not internal.
	// Otherwise, attempt to scan locally.
	scanImageFn := scanImage
	if req.containerImage.GetIsClusterLocal() {
		scanImageFn = scanImageLocal
	}

	scannedImage, err := scanWithRetries(ctx, svc, req, scanImageFn)
	if err != nil {
		// Ignore the error and set the image to something basic,
		// so alerting can progress.
		c.image = types.ToImage(req.containerImage)
		return
	}

	c.image = scannedImage.GetImage()
}

func newEnricher(cache expiringcache.Cache, serviceAccountStore store.ServiceAccountStore) *enricher {
	return &enricher{
		scanResultChan:      make(chan scanResult),
		serviceAccountStore: serviceAccountStore,
		imageCache:          cache,
		stopSig:             concurrency.NewSignal(),
	}
}

func (e *enricher) getImageFromCache(key string) (*storage.Image, bool) {
	value, _ := e.imageCache.Get(key).(*cacheValue)
	if value == nil {
		return nil, false
	}
	return value.waitAndGet(), true
}

func (e *enricher) runScan(req *scanImageRequest) imageChanResult {
	key := imagecacheutils.GetImageCacheKey(req.containerImage)

	// If the container image says that the image is not pullable, don't even bother trying to scan
	if req.containerImage.GetNotPullable() {
		return imageChanResult{
			image:        types.ToImage(req.containerImage),
			containerIdx: req.containerIdx,
		}
	}

	// Fast path
	img, ok := e.getImageFromCache(key)
	if ok {
		return imageChanResult{
			image:        img,
			containerIdx: req.containerIdx,
		}
	}

	newValue := &cacheValue{
		signal: concurrency.NewSignal(),
	}
	value := e.imageCache.GetOrSet(key, newValue).(*cacheValue)
	if newValue == value {
		value.scanAndSet(concurrency.AsContext(&e.stopSig), e.imageSvc, req)
	}
	return imageChanResult{
		image:        value.waitAndGet(),
		containerIdx: req.containerIdx,
	}
}

type scanImageRequest struct {
	containerIdx         int
	containerImage       *storage.ContainerImage
	clusterID, namespace string
	pullSecrets          []string
}

func (e *enricher) runImageScanAsync(imageChan chan<- imageChanResult, req *scanImageRequest) {
	go func() {
		// unguarded send (push to channel outside a select) is allowed because the imageChan is a buffered channel of exact size
		imageChan <- e.runScan(req)
	}()
}

func (e *enricher) getImages(deployment *storage.Deployment) []*storage.Image {
	imageChan := make(chan imageChanResult, len(deployment.GetContainers()))

	pullSecretSet := set.NewStringSet(e.serviceAccountStore.GetImagePullSecrets(deployment.GetNamespace(), deployment.GetServiceAccount())...)
	pullSecretSet.AddAll(deployment.GetImagePullSecrets()...)
	for idx, container := range deployment.GetContainers() {
		e.runImageScanAsync(imageChan, &scanImageRequest{
			containerIdx:   idx,
			containerImage: container.GetImage(),
			clusterID:      clusterid.Get(),
			namespace:      deployment.GetNamespace(),
			pullSecrets:    pullSecretSet.AsSlice(),
		})
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
