package enrichment

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
)

// UpdateScanner updates image processors map of active scanners
func (e *Enricher) UpdateScanner(scanner scannerTypes.ImageScanner) {
	e.scannerMutex.Lock()
	defer e.scannerMutex.Unlock()
	e.scanners[scanner.ProtoScanner().GetId()] = scanner
}

// RemoveScanner removes a scanner from image processors map of active scanners
func (e *Enricher) RemoveScanner(id string) {
	e.scannerMutex.Lock()
	defer e.scannerMutex.Unlock()
	delete(e.scanners, id)
}

func (e *Enricher) enrichWithScan(image *v1.Image) (bool, error) {
	e.scannerMutex.Lock()
	defer e.scannerMutex.Unlock()
	for _, scanner := range e.scanners {
		if updated, err := e.enrichImageWithScanner(image, scanner); err != nil {
			return false, err
		} else if updated {
			return true, nil
		}
	}
	return false, nil
}

// EnrichWithScanner enriches a deployment with a specific scanner.
func (e *Enricher) EnrichWithScanner(deployment *v1.Deployment, scanner scannerTypes.ImageScanner) (updated bool) {
	for _, c := range deployment.GetContainers() {
		if ok, err := e.enrichImageWithScanner(c.GetImage(), scanner); err != nil {
			logger.Error(err)
		} else if ok {
			updated = true
		}
	}

	if updated {
		e.storage.UpdateDeployment(deployment)
	}

	return
}

func (e *Enricher) equalComponents(components1, components2 []*v1.ImageScanComponents) bool {
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

	if image.GetSha() == "" {
		if _, err := e.enrichWithMetadata(image); err != nil {
			return false, err
		}
	}

	scan, err := scanner.GetLastScan(image)
	if err != nil {
		logger.Error(err)
		return false, err
	}
	if protoconv.CompareProtoTimestamps(image.GetScan().GetScanTime(), scan.GetScanTime()) != 0 || !e.equalComponents(image.GetScan().GetComponents(), scan.GetComponents()) {
		image.Scan = scan
		e.storage.UpdateImage(image)
		return true, nil
	}

	return false, nil
}
