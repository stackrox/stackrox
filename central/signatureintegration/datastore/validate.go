package datastore

import (
	"encoding/pem"
	"regexp"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
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
	if len(integration.GetCosign().GetPublicKeys()) == 0 && len(integration.GetCosignCertificates()) == 0 {
		multiErr = multierror.Append(multiErr, errors.New("integration must have at least one public key "+
			"or certificate"))
		return multiErr
	}

	if err := validateCosignKeyVerification(integration.GetCosign()); err != nil {
		multiErr = multierror.Append(multiErr, err)
	}
	if err := validateCosignCertificateVerification(integration.GetCosignCertificates()); err != nil {
		multiErr = multierror.Append(multiErr, err)
	}
	if err := validateTransparencyLogVerification(integration.GetTransparencyLog()); err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	return multiErr
}

func validateCosignKeyVerification(config *storage.CosignPublicKeyVerification) error {
	var multiErr error

	for _, publicKey := range config.GetPublicKeys() {
		if publicKey.GetName() == "" {
			err := errors.New("public key name must be filled")
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

func validateCosignCertificateVerification(configs []*storage.CosignCertificateVerification) error {
	var multiErr error

	for _, config := range configs {
		if config.GetCertificateIdentity() == "" {
			multiErr = multierror.Append(multiErr, errors.New("certificate identity must be filled"))
		}

		if _, err := regexp.Compile(config.GetCertificateIdentity()); err != nil {
			multiErr = multierror.Append(multiErr, errors.Wrap(err, "couldn't parse regex"))
		}

		if config.GetCertificateOidcIssuer() == "" {
			multiErr = multierror.Append(multiErr, errors.New("certificate issuer must be filled"))
		}

		if _, err := regexp.Compile(config.GetCertificateOidcIssuer()); err != nil {
			multiErr = multierror.Append(multiErr, errors.Wrap(err, "couldn't parse regex"))
		}

		if _, err := cryptoutils.UnmarshalCertificatesFromPEM([]byte(config.GetCertificateChainPemEnc())); err != nil {
			multiErr = multierror.Append(multiErr, errors.Wrap(err, "unmarshalling certificate chain PEM"))
		}
		if _, err := cryptoutils.UnmarshalCertificatesFromPEM([]byte(config.GetCertificatePemEnc())); err != nil {
			multiErr = multierror.Append(multiErr, errors.Wrap(err, "unmarshalling certificate PEM"))
		}

		if ctlog := config.GetCertificateTransparencyLog(); ctlog.GetEnabled() {
			if ctlogPubKey := ctlog.GetPublicKeyPemEnc(); ctlogPubKey != "" {
				ctlogKeyBlock, rest := pem.Decode([]byte(ctlogPubKey))
				if !signatures.IsValidPublicKeyPEMBlock(ctlogKeyBlock, rest) {
					multiErr = multierror.Append(multiErr, errors.New("failed to decode PEM block containing ctlog key"))
				}
			}
		}

	}

	return multiErr
}

func validateTransparencyLogVerification(config *storage.TransparencyLogVerification) error {
	if !config.GetEnabled() {
		return nil
	}

	var multiErr error

	// The Rekor URL should never be empty at this point because of the applied default value.
	// Still, we include this check to encode the expectation here.
	if !config.GetValidateOffline() && config.GetUrl() == "" {
		multiErr = multierror.Append(multiErr,
			errors.New("transparency log url must be filled when online validation is enabled"))
	}

	if rekorPubKey := config.GetPublicKeyPemEnc(); rekorPubKey != "" {
		rekorKeyBlock, rest := pem.Decode([]byte(rekorPubKey))
		if !signatures.IsValidPublicKeyPEMBlock(rekorKeyBlock, rest) {
			multiErr = multierror.Append(multiErr,
				errors.New("failed to decode PEM block containing rekor public key"))
		}
	}

	return multiErr
}
