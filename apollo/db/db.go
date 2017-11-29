package db

import (
	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
	scannerTypes "bitbucket.org/stack-rox/apollo/apollo/scanners/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// Storage is the interface for the persistent storage
type Storage interface {
	Load() error
	Close()

	AlertStorage
	BenchmarkStorage
	DeploymentStorage
	ImageRuleStorage
	ImageStorage
	RegistryStorage
	ScannerStorage
}

// AlertStorage provides storage functionality for alerts.
type AlertStorage interface {
	GetAlerts(request *v1.GetAlertsRequest) ([]*v1.Alert, error)
	AddAlert(alert *v1.Alert) error
	UpdateAlert(alert *v1.Alert) error
	RemoveAlert(id string) error
}

// BenchmarkStorage provides storage functionality for benchmarks.
type BenchmarkStorage interface {
	AddBenchmark(benchmark *v1.BenchmarkPayload)
	GetBenchmarks(request *v1.GetBenchmarksRequest) []*v1.BenchmarkPayload
}

// DeploymentStorage provides storage functionality for deployments.
type DeploymentStorage interface {
	GetDeployment(id string) (*v1.Deployment, bool, error)
	GetDeployments() ([]*v1.Deployment, error)
	AddDeployment(deployment *v1.Deployment) error
	UpdateDeployment(deployment *v1.Deployment) error
	RemoveDeployment(id string) error
}

// ImageRuleStorage provides storage functionality for image rules.
type ImageRuleStorage interface {
	GetImageRules(request *v1.GetImageRulesRequest) ([]*v1.ImageRule, error)
	AddImageRule(*v1.ImageRule) error
	UpdateImageRule(*v1.ImageRule) error
	RemoveImageRule(string) error
}

// ImageStorage provide storage functionality for images.
type ImageStorage interface {
	GetImages(request *v1.GetImagesRequest) ([]*v1.Image, error)
	AddImage(image *v1.Image) error
	UpdateImage(image *v1.Image) error
	RemoveImage(id string) error
}

// RegistryStorage provide storage functionality for registries.
type RegistryStorage interface {
	AddRegistry(name string, registry registryTypes.ImageRegistry)
	RemoveRegistry(name string)
	GetRegistries() map[string]registryTypes.ImageRegistry
}

// ScannerStorage provide storage functionality for scanner.
type ScannerStorage interface {
	AddScanner(name string, scanner scannerTypes.ImageScanner)
	RemoveScanner(name string)
	GetScanners() map[string]scannerTypes.ImageScanner
}
