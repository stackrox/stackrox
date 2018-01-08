package detection

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
)

// UpdateRegistry updates image processors map of active registries
func (d *Detector) UpdateRegistry(registry registries.ImageRegistry) {
	d.registryMutex.Lock()
	defer d.registryMutex.Unlock()
	d.registries[registry.ProtoRegistry().Name] = registry
}

// UpdateScanner updates image processors map of active scanners
func (d *Detector) UpdateScanner(scanner scannerTypes.ImageScanner) {
	d.scannerMutex.Lock()
	defer d.scannerMutex.Unlock()
	d.scanners[scanner.ProtoScanner().Name] = scanner
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
		registryMap[protoRegistry.Name] = registry
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
		scannerMap[protoScanner.Name] = scanner
	}
	d.scanners = scannerMap
	return nil
}

func (d *Detector) enrich(deployment *v1.Deployment) error {
	for _, c := range deployment.GetContainers() {
		if err := d.enrichImage(c.GetImage()); err != nil {
			return err
		}
	}

	return nil
}

func (d *Detector) enrichImage(image *v1.Image) error {
	updatedMetadata, err := d.enrichWithMetadata(image)
	if err != nil {
		return err
	}
	updatedScan, err := d.enrichWithScan(image)
	if err != nil {
		return err
	}
	if updatedMetadata || updatedScan {
		// Store image in the database
		return d.database.UpdateImage(image)
	}
	return nil
}

func (d *Detector) enrichWithMetadata(image *v1.Image) (bool, error) {
	d.registryMutex.Lock()
	defer d.registryMutex.Unlock()
	for _, registry := range d.registries {
		if !registry.Global() {
			continue
		}
		if !registry.Match(image) {
			continue
		}
		metadata, err := registry.Metadata(image)
		if err != nil {
			logger.Error(err) // This will be removed, but useful for debugging at this point
			continue
		}
		image.Metadata = metadata
		return true, nil
	}
	return false, nil
}

func (d *Detector) enrichWithScan(image *v1.Image) (bool, error) {
	d.scannerMutex.Lock()
	defer d.scannerMutex.Unlock()
	for _, scanner := range d.scanners {
		if !scanner.Global() {
			continue
		}
		if !scanner.Match(image) {
			continue
		}
		scan, err := scanner.GetLastScan(image)
		if err != nil {
			logger.Error(err)
			continue
		}
		image.Scan = scan
		return true, nil
	}
	return false, nil
}
