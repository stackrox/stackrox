package gcp

import (
	"context"

	"golang.org/x/oauth2/google"
)

// CredentialsManager manages GCP credentials based on the environment.
type CredentialsManager interface {
	Start() error
	Stop()
	GetCredentials(ctx context.Context) (*google.Credentials, error)
}

// DefaultCredentialsManager always returns the default GCP credential chain.
type DefaultCredentialsManager struct{}

var _ CredentialsManager = &DefaultCredentialsManager{}

// Start is a dummy function to fulfil the interface.
func (c *DefaultCredentialsManager) Start() error { return nil }

// Stop is a dummy function to fulfil the interface.
func (c *DefaultCredentialsManager) Stop() {}

// GetCredentials returns the default GCP credential chain.
func (c *DefaultCredentialsManager) GetCredentials(ctx context.Context) (*google.Credentials, error) {
	return google.FindDefaultCredentials(ctx)
}
