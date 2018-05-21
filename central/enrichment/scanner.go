package enrichment

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
)

func (e *Enricher) enrichWithScan(image *v1.Image) (bool, error) {
	for _, integration := range e.imageIntegrations {
		if integration.Scanner == nil {
			continue
		}
		if updated, err := e.enrichImageWithScanner(image, integration.Scanner); err != nil {
			logger.Errorf("Error enriching with scanner %s", integration.Name)
			continue
		} else if updated {
			return true, nil
		}
	}
	return false, nil
}

// enrichWithScanner enriches a deployment with a specific scanner.
func (e *Enricher) enrichWithScanner(deployment *v1.Deployment, scanner scannerTypes.ImageScanner) (updated bool) {
	for _, c := range deployment.GetContainers() {
		if ok, err := e.enrichImageWithScanner(c.GetImage(), scanner); err != nil {
			logger.Error(err)
		} else if ok {
			updated = true
		}
	}

	if updated {
		if err := e.deploymentStorage.UpdateDeployment(deployment); err != nil {
			logger.Errorf("unable to updated deployment: %s", err)
		}
	}

	return
}

func (e *Enricher) equalComponents(components1, components2 []*v1.ImageScanComponent) bool {
	if len(components1) != len(components2) {
		return false
	}
	for i := 0; i < len(components1); i++ {
		c1 := components1[i]
		c2 := components2[i]
		if len(c1.GetVulns()) != len(c2.GetVulns()) {
			return false
		}
		for j := 0; j < len(c1.GetVulns()); j++ {
			v1 := c1.GetVulns()[j]
			v2 := c2.GetVulns()[j]
			if v1.GetCve() != v2.GetCve() || v1.GetCvss() != v2.GetCvss() || v1.GetLink() != v2.GetLink() || v1.GetSummary() != v2.GetSummary() {
				return false
			}
		}
	}
	return true
}

func (e *Enricher) enrichImageWithScanner(image *v1.Image, scanner scannerTypes.ImageScanner) (bool, error) {
	if !scanner.Global() {
		return false, nil
	}
	if !scanner.Match(image) {
		return false, nil
	}
	var scan *v1.ImageScan
	scanItem := e.scanCache.Get(image.GetName().GetSha())
	if scanItem == nil {
		metrics.IncrementScanCacheMiss()
		e.scanLimiter.Wait(context.Background())

		var err error
		scan, err := scanner.GetLastScan(image)
		if err != nil {
			logger.Error(err)
			return false, err
		}
		e.scanCache.Set(image.GetName().GetSha(), scan, imageDataExpiration)
	} else {
		metrics.IncrementScanCacheHit()
		scan = scanItem.Value().(*v1.ImageScan)
	}

	if protoconv.CompareProtoTimestamps(image.GetScan().GetScanTime(), scan.GetScanTime()) != 0 || !e.equalComponents(image.GetScan().GetComponents(), scan.GetComponents()) {
		image.Scan = scan
		if err := e.imageStorage.UpdateImage(image); err != nil {
			logger.Errorf("unable to update image: %s", err)
			return false, err
		}
		return true, nil
	}

	return false, nil
}
