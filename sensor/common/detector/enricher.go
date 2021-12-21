package detector

import (
	"context"
	"strings"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/sensor/common/imagecacheutils"
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

func (c *cacheValue) scanAndSet(svc v1.ImageServiceClient, ci *storage.ContainerImage) {
	defer c.signal.Signal()

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
	if strings.Contains(ci.GetName().GetFullName(), "log4j") || strings.Contains(ci.GetName().GetFullName(), "nginx") {
		for _, c2 := range c.image.GetScan().GetComponents() {
			for _, v := range c2.GetVulns() {
				if v.GetCve() == "CVE-2009-5155" || v.GetCve() == "CVE-2021-44228" || v.GetCve() == "CVE-2007-6755" || v.GetCve() == "CVE-2004-0971" {
					log.Errorf("[scanAndSet] Pulled image from scan for image %s component %s, cve %s, state %s and snoozed? %v", c.image.GetName().GetFullName(), c2.GetName(), v.GetCve(), v.GetState().String(), v.GetSuppressed())
				}
			}
		}
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
		if strings.Contains(ci.GetName().GetFullName(), "log4j") || strings.Contains(ci.GetName().GetFullName(), "nginx") {
			log.Errorf("[runScan] NOT PULLABE image for image %s with key %+v", ci.GetName().GetFullName(), key)
		}
		return imageChanResult{
			image:        types.ToImage(ci),
			containerIdx: containerIdx,
		}
	}

	// Fast path
	img, ok := e.getImageFromCache(key)
	if ok {
		if strings.Contains(ci.GetName().GetFullName(), "log4j") || strings.Contains(ci.GetName().GetFullName(), "nginx") {
			for _, c := range img.GetScan().GetComponents() {
				for _, v := range c.GetVulns() {
					if v.GetCve() == "CVE-2009-5155" || v.GetCve() == "CVE-2021-44228" || v.GetCve() == "CVE-2007-6755" || v.GetCve() == "CVE-2004-0971" {
						log.Errorf("[runScan] Pulled image from cache for image %s with key %+v, component %s, cve %s, state %s and snoozed? %v", ci.GetName().GetFullName(), key, c.GetName(), v.GetCve(), v.GetState().String(), v.GetSuppressed())
					}
				}
			}
		}
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
		value.scanAndSet(e.imageSvc, ci)
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
	if deployment.GetName() == "vulnerable-deploy-enforce" {
		log.Errorf("Getting images from central for deploy %s", deployment.GetName())
	}
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
