package enrichment

import (
	"fmt"
	"sync"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	scannerTypes "bitbucket.org/stack-rox/apollo/pkg/scanners"
)

var (
	logger = logging.New("enrichment")
)

// Enricher enriches images with data from registries and scanners.
type Enricher struct {
	storage interface {
		db.DeploymentStorage
		db.ImageStorage
		db.RegistryStorage
		db.ScannerStorage
	}

	registryMutex sync.Mutex
	registries    map[string]registries.ImageRegistry

	scannerMutex sync.Mutex
	scanners     map[string]scannerTypes.ImageScanner
}

// New creates and returns a new Enricher.
func New(storage db.Storage) (*Enricher, error) {
	e := &Enricher{
		storage: storage,
	}
	if err := e.initializeRegistries(); err != nil {
		return nil, err
	}
	if err := e.initializeScanners(); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Enricher) initializeRegistries() error {
	protoRegistries, err := e.storage.GetRegistries(&v1.GetRegistriesRequest{})
	if err != nil {
		return err
	}
	e.registries = make(map[string]registries.ImageRegistry, len(protoRegistries))
	for _, protoRegistry := range protoRegistries {
		registry, err := registries.CreateRegistry(protoRegistry)
		if err != nil {
			return fmt.Errorf("error generating a registry from persisted registry data: %+v", err)
		}
		e.registries[protoRegistry.GetId()] = registry
	}
	return nil
}

func (e *Enricher) initializeScanners() error {
	protoScanners, err := e.storage.GetScanners(&v1.GetScannersRequest{})
	if err != nil {
		return err
	}
	e.scanners = make(map[string]scannerTypes.ImageScanner, len(protoScanners))
	for _, protoScanner := range protoScanners {
		scanner, err := scannerTypes.CreateScanner(protoScanner)
		if err != nil {
			return fmt.Errorf("error generating a scanner from persisted scanner data: %+v", err)
		}
		e.scanners[protoScanner.GetId()] = scanner
	}
	return nil
}

// Enrich enriches a deployment with data from registries and scanners.
func (e *Enricher) Enrich(deployment *v1.Deployment) (enriched bool, err error) {
	for _, c := range deployment.GetContainers() {
		if updated, err := e.enrichImage(c.GetImage()); err != nil {
			return false, err
		} else if updated {
			enriched = true
		}
	}

	if enriched {
		err = e.storage.UpdateDeployment(deployment)
	}

	return
}

func (e *Enricher) enrichImage(image *v1.Image) (bool, error) {
	updatedMetadata, err := e.enrichWithMetadata(image)
	if err != nil {
		return false, err
	}
	updatedScan, err := e.enrichWithScan(image)
	if err != nil {
		return false, err
	}
	if image.GetName().GetSha() != "" && (updatedMetadata || updatedScan) {
		// Store image in the database
		return true, e.storage.UpdateImage(image)
	}
	return false, nil
}
