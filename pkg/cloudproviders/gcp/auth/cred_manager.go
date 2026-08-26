package auth

import (
	"context"

	"golang.org/x/oauth2/google"
)

var defaultGCPAuthScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/cloud-platform.read-only",
}

// GCPAuthScopes returns the default OAuth scopes for GCP API access.
func GCPAuthScopes() []string {
	copyScopes := defaultGCPAuthScopes
	return copyScopes
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
	return google.FindDefaultCredentials(ctx, defaultGCPAuthScopes...)
}
