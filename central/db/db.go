package db

import (
	"net/http"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// Storage is the interface for the persistent storage
type Storage interface {
	BackupHandler() http.Handler
	ExportHandler() http.Handler
	Close()

	AlertStorage
	AuthProviderStorage
	BenchmarkScansStorage
	BenchmarkScheduleStorage
	BenchmarkStorage
	BenchmarkTriggerStorage
	ClusterStorage
	ImageIntegrationStorage
	LogsStorage
	DeploymentStorage
	PolicyStorage
	ImageStorage
	MultiplierStorage
	NotifierStorage
	ServiceIdentityStorage
}

// AlertStorage provides storage functionality for alerts.
type AlertStorage interface {
	GetAlert(id string) (*v1.Alert, bool, error)
	GetAlerts(request *v1.ListAlertsRequest) ([]*v1.Alert, error)
	CountAlerts() (int, error)
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
	GetBenchmark(id string) (*v1.Benchmark, bool, error)
	GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*v1.Benchmark, error)
	AddBenchmark(benchmark *v1.Benchmark) (string, error)
	UpdateBenchmark(benchmark *v1.Benchmark) error
	RemoveBenchmark(id string) error
}

// BenchmarkScheduleStorage provides storage functionality for benchmark schedules.
type BenchmarkScheduleStorage interface {
	GetBenchmarkSchedule(name string) (*v1.BenchmarkSchedule, bool, error)
	GetBenchmarkSchedules(request *v1.GetBenchmarkSchedulesRequest) ([]*v1.BenchmarkSchedule, error)
	AddBenchmarkSchedule(schedule *v1.BenchmarkSchedule) (string, error)
	UpdateBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error
	RemoveBenchmarkSchedule(name string) error
}

// BenchmarkScansStorage provides storage functionality for benchmarks scans.
type BenchmarkScansStorage interface {
	AddScan(request *v1.BenchmarkScanMetadata) error
	ListBenchmarkScans(*v1.ListBenchmarkScansRequest) ([]*v1.BenchmarkScanMetadata, error)
	GetBenchmarkScan(request *v1.GetBenchmarkScanRequest) (*v1.BenchmarkScan, bool, error)
	GetHostResults(request *v1.GetHostResultsRequest) (*v1.HostResults, bool, error)
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
	CountClusters() (int, error)
	AddCluster(cluster *v1.Cluster) (string, error)
	UpdateCluster(cluster *v1.Cluster) error
	RemoveCluster(id string) error
	UpdateClusterContactTime(id string, t time.Time) error
}

// ImageIntegrationStorage provide storage functionality for image integrations.
type ImageIntegrationStorage interface {
	GetImageIntegration(id string) (*v1.ImageIntegration, bool, error)
	GetImageIntegrations(integration *v1.GetImageIntegrationsRequest) ([]*v1.ImageIntegration, error)
	AddImageIntegration(integration *v1.ImageIntegration) (string, error)
	UpdateImageIntegration(integration *v1.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// DeploymentStorage provides storage functionality for deployments.
type DeploymentStorage interface {
	GetDeployment(id string) (*v1.Deployment, bool, error)
	GetDeployments() ([]*v1.Deployment, error)
	CountDeployments() (int, error)
	AddDeployment(deployment *v1.Deployment) error
	UpdateDeployment(deployment *v1.Deployment) error
	RemoveDeployment(id string) error
	GetTombstonedDeployments() ([]*v1.Deployment, error)
}

// ImageStorage provide storage functionality for images.
type ImageStorage interface {
	GetImage(sha string) (*v1.Image, bool, error)
	GetImages() ([]*v1.Image, error)
	CountImages() (int, error)
	AddImage(image *v1.Image) error
	UpdateImage(image *v1.Image) error
	RemoveImage(id string) error
}

// LogsStorage provide storage functionality for logs.
type LogsStorage interface {
	GetLogs() ([]string, error)
	CountLogs() (count int, err error)
	GetLogsRange() (start int64, end int64, err error)
	AddLog(log string) error
	RemoveLogs(from, to int64) error
}

// MultiplierStorage provides the storage functionality for risk scoring multipliers
type MultiplierStorage interface {
	GetMultipliers() ([]*v1.Multiplier, error)
	AddMultiplier(multiplier *v1.Multiplier) (string, error)
	UpdateMultiplier(multiplier *v1.Multiplier) error
	RemoveMultiplier(id string) error
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
	GetPolicies() ([]*v1.Policy, error)
	AddPolicy(*v1.Policy) (string, error)
	UpdatePolicy(*v1.Policy) error
	RemovePolicy(id string) error
	RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error
	DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error
}

// ServiceIdentityStorage provides storage functionality for service identities.
type ServiceIdentityStorage interface {
	GetServiceIdentities() ([]*v1.ServiceIdentity, error)
	AddServiceIdentity(identity *v1.ServiceIdentity) error
}
