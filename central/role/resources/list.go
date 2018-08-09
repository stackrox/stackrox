// Package resources lists all resource types used by Central.
package resources

import "github.com/stackrox/rox/pkg/auth/permissions"

// All resource types that we want to define (for the purposes of enforcing
// API permissions) must be defined here.
const (
	APIToken          permissions.Resource = "APIToken"
	Alert             permissions.Resource = "Alert"
	AuthProvider      permissions.Resource = "AuthProvider"
	Benchmark         permissions.Resource = "Benchmark"
	BenchmarkScan     permissions.Resource = "BenchmarkScan"
	BenchmarkSchedule permissions.Resource = "BenchmarkSchedule"
	BenchmarkTrigger  permissions.Resource = "BenchmarkTrigger"
	Cluster           permissions.Resource = "Cluster"
	DebugMetrics      permissions.Resource = "DebugMetrics"
	Deployment        permissions.Resource = "Deployment"
	DNRIntegration    permissions.Resource = "DNRIntegration"
	Image             permissions.Resource = "Image"
	ImageIntegration  permissions.Resource = "ImageIntegration"
	ImbuedLogs        permissions.Resource = "ImbuedLogs"
	Notifier          permissions.Resource = "Notifier"
	NetworkPolicy     permissions.Resource = "NetworkPolicy"
	Policy            permissions.Resource = "Policy"
	Secret            permissions.Resource = "Secret"
	ServiceIdentity   permissions.Resource = "ServiceIdentity"
)
