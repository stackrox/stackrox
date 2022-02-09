package datastore

import (
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/uuid"
)

// signatureIntegrationIDPrefix should be prepended to every human-hostile ID of a
// signature integration for readability, e.g.,
//     "io.stackrox.signatureintegration.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
const signatureIntegrationIDPrefix = "io.stackrox.signatureintegration."

// GenerateSignatureIntegrationID returns a random valid signature integration ID.
func GenerateSignatureIntegrationID() string {
	return signatureIntegrationIDPrefix + uuid.NewV4().String()
}

// ValidateSignatureIntegration checks that signature integration is valid.
func ValidateSignatureIntegration(integration *storage.SignatureIntegration) error {
	var multiErr error

	if !strings.HasPrefix(integration.GetId(), signatureIntegrationIDPrefix) {
		err := errors.Errorf("id field must be in '%s*' format", signatureIntegrationIDPrefix)
		multiErr = multierror.Append(multiErr, err)
	}
	if integration.GetName() == "" {
		multiErr = multierror.Append(multiErr, errors.New("name field must be set"))
	}
	if len(integration.GetSignatureVerificationConfigs()) == 0 {
		err := errors.New("integration must have at least one signature verification config")
		multiErr = multierror.Append(err)
	}
	for _, verificationConfig := range integration.GetSignatureVerificationConfigs() {
		switch cfg := verificationConfig.GetConfig().(type) {
		case *storage.SignatureVerificationConfig_CosignVerification:
			err := validateCosignVerification(cfg)
			if err != nil {
				multiErr = multierror.Append(multiErr, err)
			}
		default:
			// Should theoretically never happen.
			err := errox.NewErrInvariantViolation(fmt.Sprintf(
				"invalid type for signature verification config: %T", cfg))
			multiErr = multierror.Append(err)
		}
	}

	return multiErr
}

func validateCosignVerification(config *storage.SignatureVerificationConfig_CosignVerification) error {
	var multiErr error

	publicKeys := config.CosignVerification.GetPublicKeys()
	if len(publicKeys) == 0 {
		multiErr = multierror.Append(multiErr, errors.New("at least one public key should be configured for cosign verification"))
	}
	for _, publicKey := range publicKeys {
		keyBlock, rest := pem.Decode([]byte(publicKey.GetPublicKeyPemEnc()))
		if keyBlock == nil || keyBlock.Type != signatures.PublicKeyType || len(rest) > 0 {
			err := errors.Errorf("failed to decode PEM block containing public key %q", publicKey.GetName())
			multiErr = multierror.Append(multiErr, err)
		}
	}

	return multiErr
}
