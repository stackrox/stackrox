package datastore

import (
	"encoding/pem"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/uuid"
)

// GenerateSignatureIntegrationID returns a random valid signature integration ID.
func GenerateSignatureIntegrationID() string {
	return signatures.SignatureIntegrationIDPrefix + uuid.NewV4().String()
}

// ValidateSignatureIntegration checks that signature integration is valid.
func ValidateSignatureIntegration(integration *storage.SignatureIntegration) error {
	var multiErr error

	if !strings.HasPrefix(integration.GetId(), signatures.SignatureIntegrationIDPrefix) {
		err := errors.Errorf("id field must be in '%s*' format", signatures.SignatureIntegrationIDPrefix)
		multiErr = multierror.Append(multiErr, err)
	}
	if integration.GetName() == "" {
		err := errors.New("name field must be set")
		multiErr = multierror.Append(multiErr, err)
	}
	if integration.GetCosign() == nil {
		err := errors.New("integration must have at least one signature verification config")
		multiErr = multierror.Append(multiErr, err)
	} else {
		err := validateCosignVerification(integration.GetCosign())
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}
	}

	return multiErr
}

func validateCosignVerification(config *storage.CosignPublicKeyVerification) error {
	var multiErr error

	publicKeys := config.GetPublicKeys()
	if len(publicKeys) == 0 {
		err := errors.New("cosign verification must have at least one public key configured")
		multiErr = multierror.Append(multiErr, err)
	}
	for _, publicKey := range publicKeys {
		if publicKey.GetName() == "" {
			err := errors.New("public key name should be filled")
			multiErr = multierror.Append(multiErr, err)
		}

		keyBlock, rest := pem.Decode([]byte(publicKey.GetPublicKeyPemEnc()))
		if !signatures.IsValidPublicKeyPEMBlock(keyBlock, rest) {
			err := errors.Errorf("failed to decode PEM block containing public key %q", publicKey.GetName())
			multiErr = multierror.Append(multiErr, err)
		}
	}

	return multiErr
}
