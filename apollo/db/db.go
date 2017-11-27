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

	GetImages(request *v1.GetImagesRequest) ([]*v1.Image, error)
	AddImage(image *v1.Image) error
	UpdateImage(image *v1.Image) error
	RemoveImage(id string) error

	GetImageRules(request *v1.GetImageRulesRequest) ([]*v1.ImageRule, error)
	AddImageRule(*v1.ImageRule) error
	UpdateImageRule(*v1.ImageRule) error
	RemoveImageRule(string) error

	GetAlerts(request *v1.GetAlertsRequest) ([]*v1.Alert, error)
	AddAlert(alert *v1.Alert) error
	UpdateAlert(alert *v1.Alert) error
	RemoveAlert(id string) error

	AddRegistry(name string, registry registryTypes.ImageRegistry)
	RemoveRegistry(name string)
	GetRegistries() map[string]registryTypes.ImageRegistry

	AddScanner(name string, scanner scannerTypes.ImageScanner)
	RemoveScanner(name string)
	GetScanners() map[string]scannerTypes.ImageScanner

	AddBenchmark(benchmark *v1.BenchmarkPayload)
	GetBenchmarks(request *v1.GetBenchmarksRequest) []*v1.BenchmarkPayload

	DeploymentStorage
}

// DeploymentStorage provides storage functionality for deployments.
type DeploymentStorage interface {
	GetDeployment(id string) (*v1.Deployment, bool, error)
	GetDeployments() ([]*v1.Deployment, error)
	AddDeployment(deployment *v1.Deployment) error
	UpdateDeployment(deployment *v1.Deployment) error
	RemoveDeployment(id string) error
}
