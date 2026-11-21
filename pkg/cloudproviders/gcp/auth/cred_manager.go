package auth

import (
	"context"

	"golang.org/x/oauth2/google"
)

// Instead of importing the full SDK to setup auth scopes, we copied the GCP scope strings.
// Prior code (full SDK pulled into the binary):
//     artifactv1 "cloud.google.com/go/artifactregistry/apiv1"
//     securitycenterv1 "cloud.google.com/go/securitycenter/apiv1"
//     storagev1 "google.golang.org/api/storage/v1"
//     scopes := slices.Concat(
//             []string{storagev1.CloudPlatformScope},
//             artifactv1.DefaultAuthScopes(),
//             securitycenterv1.DefaultAuthScopes(),
//     )

// These are the 2 scopes from those 3 apis (artifactregistry has both, the others have only the first):
const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
const cloudPlatformScopeReadOnly = "https://www.googleapis.com/auth/cloud-platform.read-only"

var defaultAuthScopes = []string{
	cloudPlatformScope,
	cloudPlatformScopeReadOnly,
}

// DefaultAuthScopes returns the default OAuth scopes for GCP API access.
func DefaultAuthScopes() []string {
	return append([]string(nil), defaultAuthScopes...)
}

// CredentialsManager manages GCP credentials based on the environment.
//
//go:generate mockgen-wrapper
type CredentialsManager interface {
	Start()
	Stop()
	GetCredentials(ctx context.Context) (*google.Credentials, error)
}

// defaultCredentialsManager always returns the default GCP credential chain.
type defaultCredentialsManager struct{}

var _ CredentialsManager = &defaultCredentialsManager{}

// Start is a dummy function to fulfil the interface.
func (c *defaultCredentialsManager) Start() {}

// Stop is a dummy function to fulfil the interface.
func (c *defaultCredentialsManager) Stop() {}

// GetCredentials returns the default GCP credential chain.
func (c *defaultCredentialsManager) GetCredentials(ctx context.Context) (*google.Credentials, error) {
	return google.FindDefaultCredentials(ctx, cloudPlatformScope)
}
