package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

// CredentialsManager manages AWS credentials based on the environment.
type CredentialsManager interface {
	Start()
	Stop()
	NewSession(cfgs ...*aws.Config) (*session.Session, error)
}

// DefaultCredentialsManager always returns the default AWS credential chain.
type DefaultCredentialsManager struct{}

var _ CredentialsManager = &DefaultCredentialsManager{}

// Start is a dummy function to fulfil the interface.
func (c *DefaultCredentialsManager) Start() {}

// Stop is a dummy function to fulfil the interface.
func (c *DefaultCredentialsManager) Stop() {}

// NewSession creates a new AWS session based on the default AWS credential chain.
func (c *DefaultCredentialsManager) NewSession(cfgs ...*aws.Config) (*session.Session, error) {
	opts := session.Options{}
	opts.Config.MergeIn(cfgs...)
	return session.NewSessionWithOptions(opts)
}
