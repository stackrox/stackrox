package signatures

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
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
	image *storage.Image) (storage.ImageSignatureVerificationResult_Status, []string, error) {
	// Short circuit if we do not have any public keys configured to verify against.
	if len(c.parsedPublicKeys) == 0 {
		return storage.ImageSignatureVerificationResult_FAILED_VERIFICATION, nil, errNoKeysToVerifyAgainst
	}

	var opts cosign.CheckOpts

	// By default, verify the claim within the payload that is specified with the simple signing format.
	// Right now, we are not supporting any additional annotations within the claim.
	opts.ClaimVerifier = cosign.SimpleClaimVerifier

	// With the latest version of cosign, by default signatures will be uploaded to rekor. This means that also during
	// verification, an entry in the transparency log will be expected and verified.
	// Since currently this is not the case in what we support / offer, explicitly disable this for now until we enable
	// and expect this to be the case.
	opts.IgnoreTlog = true
	opts.IgnoreSCT = true

	sigs, hash, err := retrieveVerificationDataFromImage(image)
	if err != nil {
		return getVerificationResultStatusFromErr(err), nil, err
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
		verifiedImageReferences, err := verifyImageSignatures(ctx, sigs, hash, image, opts)
		if err == nil {
			if len(verifiedImageReferences) == 0 {
				log.Infof("no verified image references found, defaulting to default image name %q", image.GetName().GetFullName())
				// Fallback to the default name of the image if the reference is empty.
				verifiedImageReferences = []string{image.GetName().GetFullName()}
			}
			return storage.ImageSignatureVerificationResult_VERIFIED, verifiedImageReferences, nil
		}
		allVerifyErrs = multierror.Append(allVerifyErrs, err)
	}

	return storage.ImageSignatureVerificationResult_FAILED_VERIFICATION, nil, allVerifyErrs
}

func verifyImageSignatures(ctx context.Context, signatures []oci.Signature, imageHash gcrv1.Hash, image *storage.Image,
	cosignOpts cosign.CheckOpts) (verifiedImageReferences []string, verificationErrors error) {
	for _, signature := range signatures {
		// The bundle references a rekor bundle within the transparency log. Since we do not support this, the
		// bundle verified will _always_ be false.
		// See: https://github.com/sigstore/cosign/blob/eaee4b7da0c1a42326bd82c6a4da7e16741db266/pkg/cosign/verify.go#L584-L586.
		// If there is no error during the verification, the signature was successfully verified
		// as well as the claims.
		_, err := cosign.VerifyImageSignature(ctx, signature, imageHash, &cosignOpts)

		if err != nil {
			verificationErrors = multierror.Append(verificationErrors, err)
			continue
		}

		if verifiedImageReferences, err = getVerifiedImageReference(signature, image); err != nil {
			verificationErrors = multierror.Append(verificationErrors, err)
		}
	}
	return verifiedImageReferences, verificationErrors
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

// getVerifiedImageReferenceFromSignature retrieves the verified docker reference in the format of
// <registry>/<repository> from the payload of the oci.Signature and filters out image names that are verified by
// the docker reference using the image names associated with the storage.Image.
func getVerifiedImageReference(signature oci.Signature, image *storage.Image) ([]string, error) {
	payloadBytes, err := signature.Payload()
	if err != nil {
		return nil, err
	}
	// The payload of each signature will be the JSON representation of the simple signing format.
	// This will include the docker manifest reference which was used for this specific signature, which will be our
	// reference which is valid for this specific signature.
	var simpleContainer payload.SimpleContainerImage
	if err := json.Unmarshal(payloadBytes, &simpleContainer); err != nil {
		return nil, err
	}

	// Match all image names that share the same registry and repository for the docker reference of the signature.
	// This will ensure we mark each image name as verified as long as it is within:
	// - the same registry
	// - the same repository
	// - and has the same digest
	// This way we also cover the case where we e.g. reference an image with digest format (<registry>/<repository>@<digest>)
	// as well as images using floating tags (<registry>/<repository>:<tag>).
	signatureImageReference := simpleContainer.Critical.Identity.DockerReference
	log.Debugf("Retrieving verified image references from the image names [%v] and image reference within the "+
		"signature %q", image.GetNames(), signatureImageReference)
	var verifiedImageReferences []string
	for _, name := range image.GetNames() {
		reference, err := dockerReferenceFromImageName(name)
		if err != nil {
			// Theoretically, all references should be parsable.
			// In case we somehow get an invalid entry, we will log the occurrence and skip this entry.
			log.Errorf("Failed to retrieve the reference for image name %s: %v", name.GetFullName(), err)
			continue
		}
		if signatureImageReference == reference {
			verifiedImageReferences = append(verifiedImageReferences, name.GetFullName())
		}
	}
	return verifiedImageReferences, nil
}

func dockerReferenceFromImageName(imageName *storage.ImageName) (string, error) {
	ref, err := name.ParseReference(imageName.GetFullName())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", ref.Context().Registry.RegistryStr(), ref.Context().RepositoryStr()), nil
}
