package signatures

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"github.com/google/go-containerregistry/pkg/name"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/oci"
	"github.com/sigstore/cosign/pkg/oci/static"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

const publicKeyType = "PUBLIC KEY"

type publicKeyVerifier struct {
	parsedPublicKeys []crypto.PublicKey
}

// newPublicKeyVerifier creates a public key verifier with the given configuration.
// It will return an error if the provided public keys could not be parsed or the base64 decoding failed.
func newPublicKeyVerifier(config *storage.SignatureVerificationConfig_PublicKey) (*publicKeyVerifier, error) {
	base64EncPublicKeys := config.PublicKey.GetPublicKeysBase64Enc()

	parsedKeys := make([]crypto.PublicKey, 0, len(base64EncPublicKeys))
	for _, base64EncKey := range base64EncPublicKeys {
		// Each key should be base64 encoded.
		decodedKey, err := base64.StdEncoding.DecodeString(base64EncKey)
		if err != nil {
			return nil, errors.Wrap(err, "decoding base64 encoded key")
		}

		// We expect the key to be PEM encoded. There should be no rest returned after decoding.
		keyBlock, rest := pem.Decode(decodedKey)
		if keyBlock == nil || keyBlock.Type != publicKeyType || len(rest) > 0 {
			return nil, errorhelpers.NewErrInvariantViolation(
				"failed to decode PEM block containing public key")
		}

		parsedKey, err := x509.ParsePKIXPublicKey(keyBlock.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "parsing DER encoded public key")
		}
		parsedKeys = append(parsedKeys, parsedKey)
	}

	return &publicKeyVerifier{parsedPublicKeys: parsedKeys}, nil
}

func retrieveVerificationDataFromImage(img *storage.Image) ([]oci.Signature, gcrv1.Hash, error) {
	// The hash is required for claim verification.
	hash, err := gcrv1.NewHash(img.GetMetadata().GetV1().GetDigest())
	if err != nil {
		return nil, gcrv1.Hash{}, errors.Wrap(err, "creating hash")
	}

	// We need a digest to create the payload.
	digest, err := name.NewDigest(img.GetMetadata().GetV1().GetDigest())
	if err != nil {
		return nil, gcrv1.Hash{}, errors.Wrap(err, "creating digest")
	}

	// TODO(dhaus): Double check, if the payload is not / should not be a part of what we save on the image proto.
	// The payload will be attached to the signature for claim verification.
	sigPayload, err := payload.Cosign{Image: digest}.MarshalJSON()
	if err != nil {
		return nil, gcrv1.Hash{}, errors.Wrap(err, "creating signature payload")
	}

	// Each signature contains the base64 encoded version of it and the associated payload.
	// In the future, this will also include potential rekor bundles for keyless verification.
	signatures := make([]oci.Signature, 0, len(img.GetSignature().GetSignatures()))
	for _, imgSig := range img.GetSignature().GetSignatures() {
		sig, err := static.NewSignature(sigPayload, imgSig.GetCosign().GetRawSignatureBase64Enc())
		if err != nil {
			return nil, gcrv1.Hash{}, errors.Wrap(err, "creating OCI signatures")
		}
		signatures = append(signatures, sig)
	}

	return signatures, hash, nil
}

// VerifySignature implements the SignatureVerifier interface.
// TODO: Right now only a stub implementation for the first abstraction.
func (c *publicKeyVerifier) VerifySignature(rawSignature []byte) (storage.ImageSignatureVerificationResult_Status, error) {
	opts := &cosign.CheckOpts{}
	ctx := context.Background()
	// By default, verify the claim within the payload that is specified with the simple signing format.
	// Right now, we are not supporting any additional annotations within the claim.
	opts.ClaimVerifier = cosign.SimpleClaimVerifier

	errList := errorhelpers.NewErrorList("public key signature verification")

	// TODO(dhaus): Replace with the storage.Image once the func signature has been changed.
	sigs, hash, err := retrieveVerificationDataFromImage(nil)
	if err != nil {
		return storage.ImageSignatureVerificationResult_CORRUPTED_SIGNATURE, err
	}

	for _, key := range c.parsedPublicKeys {
		// For now, only supporting SHA256 as algorithm.
		v, err := signature.LoadVerifier(key, crypto.SHA256)
		if err != nil {
			errList.AddError(err)
			// TODO(dhaus): What is the consensus here from a user point of view? Should we display the "highest priority"
			// error, or the most common? In the case of multiple keys, we should potentially be able to provide a
			// key based verification error (potentially). Otherwise, this is really weird.
			continue
		}
		opts.SigVerifier = v
		for _, sig := range sigs {
			// The bundle references a rekor bundle within the transparency log. Since we do not support this, the
			// bundle verified will _always_ be false.
			// See: https://github.com/sigstore/cosign/blob/eaee4b7da0c1a42326bd82c6a4da7e16741db266/pkg/cosign/verify.go#L584-L586.
			// If there is no error during the verification, the signature was successfully verified as well as the claims.
			_, err = cosign.VerifyImageSignature(ctx, sig, hash, opts)

			// Short circuit on the first public key that successfully verified the signature, since they are bundled within
			// a signature integration.
			// TODO(dhaus): In the future, we can return a list of public keys that successfully verified the signature.
			if err == nil {
				return storage.ImageSignatureVerificationResult_VERIFIED, nil
			}
			errList.AddError(err)
		}
	}

	return storage.ImageSignatureVerificationResult_FAILED_VERIFICATION, errList.ToError()
}
