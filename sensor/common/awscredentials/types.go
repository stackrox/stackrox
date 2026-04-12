package awscredentials

import (
	"github.com/stackrox/rox/pkg/docker/config"
	"time"
)

// RegistryCredentials carries credential information to access AWS-based
// registries.
type RegistryCredentials struct {
	AWSAccount   string
	AWSRegion    string
	DockerConfig *config.DockerConfigEntry
	ExpirestAt   time.Time
}

// RegistryCredentialsManager is a sensor component that manages
// credentials for docker registries.
//
//go:generate mockgen-wrapper
type RegistryCredentialsManager interface {
	// GetRegistryCredentials returns the most recent registry credential for the given
	// registry URI, or `nil` if not available.
	GetRegistryCredentials(r string) *RegistryCredentials
	Start()
	Stop()
}
