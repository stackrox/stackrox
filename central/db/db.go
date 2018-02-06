package db

import (
	"net/http"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// Storage is the interface for the persistent storage
type Storage interface {
	BackupHandler() http.Handler
	Close()

	AlertStorage
	AuthProviderStorage
	BenchmarkScansStorage
	BenchmarkScheduleStorage
	BenchmarkStorage
	BenchmarkTriggerStorage
	ClusterStorage
	DeploymentStorage
	PolicyStorage
	ImageStorage
	NotifierStorage
	RegistryStorage
	ScannerStorage
	ServiceIdentityStorage
}

// AlertStorage provides storage functionality for alerts.
type AlertStorage interface {
	GetAlert(id string) (*v1.Alert, bool, error)
	GetAlerts(request *v1.GetAlertsRequest) ([]*v1.Alert, error)
	AddAlert(alert *v1.Alert) error
	UpdateAlert(alert *v1.Alert) error
	RemoveAlert(id string) error
}

// AuthProviderStorage provide storage functionality for authProvider.
type AuthProviderStorage interface {
	GetAuthProvider(id string) (*v1.AuthProvider, bool, error)
	GetAuthProviders(request *v1.GetAuthProvidersRequest) ([]*v1.AuthProvider, error)
	AddAuthProvider(authProvider *v1.AuthProvider) (string, error)
	UpdateAuthProvider(authProvider *v1.AuthProvider) error
	RemoveAuthProvider(id string) error
}

// BenchmarkStorage provides storage functionality for benchmarks results.
type BenchmarkStorage interface {
	GetBenchmark(name string) (*v1.Benchmark, bool, error)
	GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*v1.Benchmark, error)
	AddBenchmark(benchmark *v1.Benchmark) error
	UpdateBenchmark(benchmark *v1.Benchmark) error
	RemoveBenchmark(name string) error
}

// BenchmarkScheduleStorage provides storage functionality for benchmark schedules.
type BenchmarkScheduleStorage interface {
	GetBenchmarkSchedule(name string) (*v1.BenchmarkSchedule, bool, error)
	GetBenchmarkSchedules(request *v1.GetBenchmarkSchedulesRequest) ([]*v1.BenchmarkSchedule, error)
	AddBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error
	UpdateBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error
	RemoveBenchmarkSchedule(name string) error
}

// BenchmarkScansStorage provides storage functionality for benchmarks scans.
type BenchmarkScansStorage interface {
	AddScan(request *v1.BenchmarkScanMetadata) error
	ListBenchmarkScans(*v1.ListBenchmarkScansRequest) ([]*v1.BenchmarkScanMetadata, error)
	GetBenchmarkScan(request *v1.GetBenchmarkScanRequest) (*v1.BenchmarkScan, bool, error)
	AddBenchmarkResult(benchmark *v1.BenchmarkResult) error
}

// BenchmarkTriggerStorage provides storage functionality for benchmarks triggers.
type BenchmarkTriggerStorage interface {
	GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*v1.BenchmarkTrigger, error)
	AddBenchmarkTrigger(trigger *v1.BenchmarkTrigger) error
}

// ClusterStorage provides storage functionality for clusters.
type ClusterStorage interface {
	GetCluster(id string) (*v1.Cluster, bool, error)
	GetClusters() ([]*v1.Cluster, error)
	AddCluster(cluster *v1.Cluster) (string, error)
	UpdateCluster(cluster *v1.Cluster) error
	RemoveCluster(id string) error
	UpdateClusterContactTime(id string, t time.Time) error
}

// DeploymentStorage provides storage functionality for deployments.
type DeploymentStorage interface {
	GetDeployment(id string) (*v1.Deployment, bool, error)
	GetDeployments(request *v1.GetDeploymentsRequest) ([]*v1.Deployment, error)
	AddDeployment(deployment *v1.Deployment) error
	UpdateDeployment(deployment *v1.Deployment) error
	RemoveDeployment(id string) error
}

// ImageStorage provide storage functionality for images.
type ImageStorage interface {
	GetImage(sha string) (*v1.Image, bool, error)
	GetImages(request *v1.GetImagesRequest) ([]*v1.Image, error)
	AddImage(image *v1.Image) error
	UpdateImage(image *v1.Image) error
	RemoveImage(id string) error
}

// NotifierStorage provide storage functionality for notifiers
type NotifierStorage interface {
	GetNotifier(id string) (*v1.Notifier, bool, error)
	GetNotifiers(request *v1.GetNotifiersRequest) ([]*v1.Notifier, error)
	AddNotifier(notifier *v1.Notifier) (string, error)
	UpdateNotifier(notifier *v1.Notifier) error
	RemoveNotifier(id string) error
}

// PolicyStorage provides storage functionality for policies.
type PolicyStorage interface {
	GetPolicy(id string) (*v1.Policy, bool, error)
	GetPolicies(request *v1.GetPoliciesRequest) ([]*v1.Policy, error)
	AddPolicy(*v1.Policy) (string, error)
	UpdatePolicy(*v1.Policy) error
	RemovePolicy(id string) error
}

// RegistryStorage provide storage functionality for registries.
type RegistryStorage interface {
	GetRegistry(id string) (*v1.Registry, bool, error)
	GetRegistries(request *v1.GetRegistriesRequest) ([]*v1.Registry, error)
	AddRegistry(registry *v1.Registry) (string, error)
	UpdateRegistry(registry *v1.Registry) error
	RemoveRegistry(id string) error
}

// ScannerStorage provide storage functionality for scanner.
type ScannerStorage interface {
	GetScanner(id string) (*v1.Scanner, bool, error)
	GetScanners(request *v1.GetScannersRequest) ([]*v1.Scanner, error)
	AddScanner(scanner *v1.Scanner) (string, error)
	UpdateScanner(scanner *v1.Scanner) error
	RemoveScanner(id string) error
}

// ServiceIdentityStorage provides storage functionality for service identities.
type ServiceIdentityStorage interface {
	GetServiceIdentities() ([]*v1.ServiceIdentity, error)
	AddServiceIdentity(identity *v1.ServiceIdentity) error
}
