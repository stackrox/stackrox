package signatures

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/oci"
	"github.com/sigstore/cosign/pkg/oci/static"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errox"
	imgUtils "github.com/stackrox/stackrox/pkg/images/utils"
)

const (
	publicKeyType = "PUBLIC KEY"
	sha256Algo    = "sha256"
)

var (
	errNoImageSHA            = errors.New("no image SHA found")
	errInvalidHashAlgo       = errox.InvalidArgs.New("invalid hash algorithm used")
	errNoKeysToVerifyAgainst = errors.New("no keys to verify against")
	errHashCreation          = errox.InvariantViolation.New("creating hash")
	errCorruptedSignature    = errox.InvariantViolation.New("corrupted signature")
)

type cosignPublicKeyVerifier struct {
	parsedPublicKeys []crypto.PublicKey
}

var _ SignatureVerifier = (*cosignPublicKeyVerifier)(nil)

// IsValidPublicKeyPEMBlock is a helper function which checks whether public key PEM block was successfully decoded.
func IsValidPublicKeyPEMBlock(keyBlock *pem.Block, rest []byte) bool {
	return keyBlock != nil && keyBlock.Type == publicKeyType && len(rest) == 0
}

// newCosignPublicKeyVerifier creates a public key verifier with the given Cosign configuration. The provided public keys
// MUST be valid PEM encoded ones.
// It will return an error if the provided public keys could not be parsed.
func newCosignPublicKeyVerifier(config *storage.CosignPublicKeyVerification) (*cosignPublicKeyVerifier, error) {
	publicKeys := config.GetPublicKeys()
	parsedKeys := make([]crypto.PublicKey, 0, len(publicKeys))
	for _, publicKey := range publicKeys {
		// We expect the key to be PEM encoded. There should be no rest returned after decoding.
		keyBlock, rest := pem.Decode([]byte(publicKey.GetPublicKeyPemEnc()))
		if !IsValidPublicKeyPEMBlock(keyBlock, rest) {
			return nil, errox.InvariantViolation.Newf("failed to decode PEM block containing public key %q", publicKey.GetName())
		}

		parsedKey, err := x509.ParsePKIXPublicKey(keyBlock.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "parsing DER encoded public key")
		}
		parsedKeys = append(parsedKeys, parsedKey)
	}

	return &cosignPublicKeyVerifier{parsedPublicKeys: parsedKeys}, nil
}

// VerifySignature implements the SignatureVerifier interface.
// The signature of the image will be verified using cosign. It will include the verification via public key
// as well as the claim verification of the payload of the signature.
func (c *cosignPublicKeyVerifier) VerifySignature(ctx context.Context,
	image *storage.Image) (storage.ImageSignatureVerificationResult_Status, error) {
	// Short circuit if we do not have any public keys configured to verify against.
	if len(c.parsedPublicKeys) == 0 {
		return storage.ImageSignatureVerificationResult_FAILED_VERIFICATION, errNoKeysToVerifyAgainst
	}

	var opts cosign.CheckOpts

	// By default, verify the claim within the payload that is specified with the simple signing format.
	// Right now, we are not supporting any additional annotations within the claim.
	opts.ClaimVerifier = cosign.SimpleClaimVerifier

	sigs, hash, err := retrieveVerificationDataFromImage(image)
	if err != nil {
		return getVerificationResultStatusFromErr(err), err
	}

	var allVerifyErrs error
	for _, key := range c.parsedPublicKeys {
		// For now, only supporting SHA256 as algorithm.
		v, err := signature.LoadVerifier(key, crypto.SHA256)
		if err != nil {
			allVerifyErrs = multierror.Append(allVerifyErrs, errors.Wrap(err, "creating verifier"))
			continue
		}
		opts.SigVerifier = v
		for _, sig := range sigs {
			// The bundle references a rekor bundle within the transparency log. Since we do not support this, the
			// bundle verified will _always_ be false.
			// See: https://github.com/sigstore/cosign/blob/eaee4b7da0c1a42326bd82c6a4da7e16741db266/pkg/cosign/verify.go#L584-L586.
			// If there is no error during the verification, the signature was successfully verified
			// as well as the claims.
			_, err = cosign.VerifyImageSignature(ctx, sig, hash, &opts)

			// Short circuit on the first public key that successfully verified the signature, since they are bundled
			// within a single signature integration.
			if err == nil {
				return storage.ImageSignatureVerificationResult_VERIFIED, nil
			}
			allVerifyErrs = multierror.Append(allVerifyErrs, err)
		}
	}

	return storage.ImageSignatureVerificationResult_FAILED_VERIFICATION, allVerifyErrs
}

// getVerificationResultStatusFromErr will map an error to a specific storage.ImageSignatureVerificationResult_Status.
// This is done in an effort to return appropriate status codes to the client triggering the signature verification.
func getVerificationResultStatusFromErr(err error) storage.ImageSignatureVerificationResult_Status {
	if errors.Is(err, errInvalidHashAlgo) {
		return storage.ImageSignatureVerificationResult_INVALID_SIGNATURE_ALGO
	}

	if errors.Is(err, errCorruptedSignature) {
		return storage.ImageSignatureVerificationResult_CORRUPTED_SIGNATURE
	}

	return storage.ImageSignatureVerificationResult_GENERIC_ERROR
}

func retrieveVerificationDataFromImage(image *storage.Image) ([]oci.Signature, gcrv1.Hash, error) {
	imgSHA := imgUtils.GetSHA(image)
	// If there is no digest associated with the image, we cannot safely do signature and claim verification.
	if imgSHA == "" {
		return nil, gcrv1.Hash{}, errNoImageSHA
	}

	// The hash is required for claim verification.
	hash, err := gcrv1.NewHash(imgSHA)
	if err != nil {
		return nil, gcrv1.Hash{}, errHashCreation.New(err.Error())
	}

	// Theoretically, this should never happen, as gcrv1.NewHash _currently_ doesn't support any other hash algorithm.
	// See: https://github.com/google/go-containerregistry/blob/main/pkg/v1/hash.go#L78
	// We should keep this check although, in case there are changes in the library.
	if hash.Algorithm != sha256Algo {
		return nil, gcrv1.Hash{}, errInvalidHashAlgo.Newf(
			"invalid hashing algorithm %s used, only SHA256 is supported", hash.Algorithm)
	}

	// Each signature contains the base64 encoded version of it and the associated payload.
	// In the future, this will also include potential rekor bundles for keyless verification.
	signatures := make([]oci.Signature, 0, len(image.GetSignature().GetSignatures()))
	for _, imgSig := range image.GetSignature().GetSignatures() {
		if imgSig.GetCosign() == nil {
			continue
		}
		b64Sig := base64.StdEncoding.EncodeToString(imgSig.GetCosign().GetRawSignature())

		sig, err := static.NewSignature(imgSig.GetCosign().GetSignaturePayload(), b64Sig)
		if err != nil {
			// Theoretically, this error should never happen, as the only error currently occurs when using options,
			// which we do not use _yet_. When introducing support for rekor bundles, this could potentially error.
			return nil, gcrv1.Hash{}, errCorruptedSignature.CausedBy(err)
		}
		signatures = append(signatures, sig)
	}

	return signatures, hash, nil
}
