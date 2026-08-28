package auth

import (
	"context"

	"golang.org/x/oauth2/google"
)

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
