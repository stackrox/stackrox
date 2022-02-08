package datastore

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
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

// IsValidSignatureIntegrationID checks if id is in correct format.
func IsValidSignatureIntegrationID(id string) bool {
	return strings.HasPrefix(id, signatureIntegrationIDPrefix)
}

func ValidateSignatureIntegration(integration *storage.SignatureIntegration) error {
	var multiErr error

	if !IsValidSignatureIntegrationID(integration.GetId()) {
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
		cosignVerification := verificationConfig.GetCosignVerification()
		if cosignVerification != nil {
			publicKeys := cosignVerification.GetPublicKeys()
			for index, publicKey := range publicKeys {
				if publicKey.GetName() == "" {
					return errorhelpers.NewErrInvalidArgs(fmt.Sprintf("publicKeys[%d] has no name assigned", index))
				}
				if _, err := base64.StdEncoding.DecodeString(publicKey.GetPublicKeysBase64Enc()); err != nil {
					return errorhelpers.NewErrInvalidArgs(fmt.Sprintf("public key %q has invalid base64 encoding", publicKey.GetName()))
				}
			}
		}
	}

	return multiErr
}
