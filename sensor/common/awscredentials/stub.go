//go:build !awsecr

package awscredentials

import "github.com/pkg/errors"

// NewECRCredentialsManager returns an error when the awsecr build tag is not set.
// Build with -tags awsecr to enable AWS ECR credential management.
func NewECRCredentialsManager(_ string) (RegistryCredentialsManager, error) {
	return nil, errors.New("ECR credential manager not available (built without awsecr tag)")
}
