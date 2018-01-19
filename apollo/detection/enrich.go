package detection

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
)

// UpdateRegistry updates image processors map of active registries
func (d *Detector) UpdateRegistry(registry registries.ImageRegistry) {
	d.registryMutex.Lock()
	defer d.registryMutex.Unlock()
	d.registries[registry.ProtoRegistry().GetId()] = registry
	go d.reprocessRegistry(registry)
}

// RemoveRegistry removes a registry from image processors map of active registries
func (d *Detector) RemoveRegistry(id string) {
	d.registryMutex.Lock()
	defer d.registryMutex.Unlock()
	delete(d.registries, id)
}

// UpdateScanner updates image processors map of active scanners
func (d *Detector) UpdateScanner(scanner scannerTypes.ImageScanner) {
	d.scannerMutex.Lock()
	defer d.scannerMutex.Unlock()
	d.scanners[scanner.ProtoScanner().GetId()] = scanner
	go d.reprocessScanner(scanner)
}

// RemoveScanner removes a scanner from image processors map of active scanners
func (d *Detector) RemoveScanner(id string) {
	d.scannerMutex.Lock()
	defer d.scannerMutex.Unlock()
	delete(d.scanners, id)
}

func (d *Detector) initializeRegistries() error {
	registryMap := make(map[string]registries.ImageRegistry)
	protoRegistries, err := d.database.GetRegistries(&v1.GetRegistriesRequest{})
	if err != nil {
		return err
	}
	for _, protoRegistry := range protoRegistries {
		registry, err := registries.CreateRegistry(protoRegistry)
		if err != nil {
			return fmt.Errorf("error generating a registry from persisted registry data: %+v", err)
		}
		registryMap[protoRegistry.GetId()] = registry
	}
	d.registries = registryMap
	return nil
}

func (d *Detector) initializeScanners() error {
	scannerMap := make(map[string]scannerTypes.ImageScanner)
	protoScanners, err := d.database.GetScanners(&v1.GetScannersRequest{})
	if err != nil {
		return err
	}
	for _, protoScanner := range protoScanners {
		scanner, err := scanners.CreateScanner(protoScanner)
		if err != nil {
			return fmt.Errorf("error generating a registry from persisted registry data: %+v", err)
		}
		scannerMap[protoScanner.GetId()] = scanner
	}
	d.scanners = scannerMap
	return nil
}

func (d *Detector) enrich(deployment *v1.Deployment) (enriched bool, err error) {
	for _, c := range deployment.GetContainers() {
		if updated, err := d.enrichImage(c.GetImage()); err != nil {
			return false, err
		} else if updated {
			enriched = true
		}
	}

	return
}

func (d *Detector) enrichImage(image *v1.Image) (bool, error) {
	updatedMetadata, err := d.enrichWithMetadata(image)
	if err != nil {
		return false, err
	}
	updatedScan, err := d.enrichWithScan(image)
	if err != nil {
		return false, err
	}
	if updatedMetadata || updatedScan {
		// Store image in the database
		return true, d.database.UpdateImage(image)
	}
	return false, nil
}

func (d *Detector) enrichWithMetadata(image *v1.Image) (updated bool, err error) {
	d.registryMutex.Lock()
	defer d.registryMutex.Unlock()
	for _, registry := range d.registries {
		if updated, err = d.enrichImageWithRegistry(image, registry); err != nil {
			return
		} else if updated {
			return
		}
	}
	return
}

func (d *Detector) enrichWithRegistry(deployment *v1.Deployment, registry registries.ImageRegistry) (updated bool) {
	for _, c := range deployment.GetContainers() {
		if ok, err := d.enrichImageWithRegistry(c.GetImage(), registry); err != nil {
			logger.Error(err)
		} else if ok {
			updated = true
		}
	}

	return
}

func (d *Detector) enrichImageWithRegistry(image *v1.Image, registry registries.ImageRegistry) (bool, error) {
	if !registry.Global() {
		return false, nil
	}
	if !registry.Match(image) {
		return false, nil
	}
	metadata, err := registry.Metadata(image)
	if err != nil {
		logger.Error(err)
		return false, err
	}

	if protoconv.CompareProtoTimestamps(image.GetMetadata().GetCreated(), metadata.GetCreated()) != 0 {
		image.Metadata = metadata
		return true, nil
	}

	return false, nil
}

func (d *Detector) enrichWithScan(image *v1.Image) (bool, error) {
	d.scannerMutex.Lock()
	defer d.scannerMutex.Unlock()
	for _, scanner := range d.scanners {
		if updated, err := d.enrichImageWithScanner(image, scanner); err != nil {
			return false, err
		} else if updated {
			return true, nil
		}
	}
	return false, nil
}

func (d *Detector) enrichWithScanner(deployment *v1.Deployment, scanner scannerTypes.ImageScanner) (updated bool) {
	for _, c := range deployment.GetContainers() {
		if ok, err := d.enrichImageWithScanner(c.GetImage(), scanner); err != nil {
			logger.Error(err)
		} else if ok {
			updated = true
		}
	}

	return
}

func (d *Detector) enrichImageWithScanner(image *v1.Image, scanner scannerTypes.ImageScanner) (bool, error) {
	if !scanner.Global() {
		return false, nil
	}
	if !scanner.Match(image) {
		return false, nil
	}

	if image.GetSha() == "" {
		if _, err := d.enrichWithMetadata(image); err != nil {
			return false, err
		}
	}

	scan, err := scanner.GetLastScan(image)
	if err != nil {
		logger.Error(err)
		return false, err
	}
	if protoconv.CompareProtoTimestamps(image.GetScan().GetScanTime(), scan.GetScanTime()) != 0 {
		image.Scan = scan
		return true, nil
	}

	return false, nil
}
