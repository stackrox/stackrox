package signatures

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	gcrRemote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/oci"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/sigstore/cosign/pkg/oci/static"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
)

const (
	publicKeyType = "PUBLIC KEY"
	sha256Algo    = "sha256"
)

var (
	errNoImageSHA            = errors.New("no image SHA found")
	errInvalidHashAlgo       = errors.New("invalid hash algorithm used")
	errNoKeysToVerifyAgainst = errors.New("no keys to verify against")
	errHashCreation          = errors.New("creating hash")
	errCorruptedSignature    = errors.New("corrupted signature")
)

type cosignPublicKey struct {
	parsedPublicKeys []crypto.PublicKey
}

var _ SignatureVerifier = (*cosignPublicKey)(nil)
var _ SignatureFetcher = (*cosignPublicKey)(nil)

// IsValidPublicKeyPEMBlock is a helper function which checks whether public key PEM block was successfully decoded.
func IsValidPublicKeyPEMBlock(keyBlock *pem.Block, rest []byte) bool {
	return keyBlock != nil && keyBlock.Type == publicKeyType && len(rest) == 0
}

// newCosignPublicKeyVerifier creates a public key verifier with the given Cosign configuration. The provided public keys
// MUST be valid PEM encoded ones.
// It will return an error if the provided public keys could not be parsed.
func newCosignPublicKeyVerifier(config *storage.CosignPublicKeyVerification) (*cosignPublicKey, error) {
	publicKeys := config.GetPublicKeys()
	parsedKeys := make([]crypto.PublicKey, 0, len(publicKeys))
	for _, publicKey := range publicKeys {
		// We expect the key to be PEM encoded. There should be no rest returned after decoding.
		keyBlock, rest := pem.Decode([]byte(publicKey.GetPublicKeyPemEnc()))
		if !IsValidPublicKeyPEMBlock(keyBlock, rest) {
			return nil, errox.Newf(errox.InvariantViolation, "failed to decode PEM block containing public key %q", publicKey.GetName())
		}

		parsedKey, err := x509.ParsePKIXPublicKey(keyBlock.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "parsing DER encoded public key")
		}
		parsedKeys = append(parsedKeys, parsedKey)
	}

	return &cosignPublicKey{parsedPublicKeys: parsedKeys}, nil
}

// VerifySignature implements the SignatureVerifier interface.
// The signature of the image will be verified using cosign. It will include the verification via public key
// as well as the claim verification of the payload of the signature.
func (c *cosignPublicKey) VerifySignature(ctx context.Context,
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

// FetchSignature implements the SignatureFetcher interface.
// The signature associated with the image will be fetched from the given registry.
// It will return the storage.ImageSignature and a boolean, indicating whether any signatures were found or not.
func (c *cosignPublicKey) FetchSignature(ctx context.Context, image *storage.Image,
	registry registryTypes.ImageRegistry) (*storage.ImageSignature, bool) {
	// Since cosign makes heavy use of google/go-containerregistry, we need to parse the image's full name as a
	// name.Reference.
	imgFullName := image.GetName().GetFullName()
	imgRef, err := name.ParseReference(imgFullName)
	if err != nil {
		log.Errorf("Parsing image reference %q: %v", imgFullName, err)
		return nil, false
	}

	// Fetch the signatures by injecting the registry specific authentication options to the google/go-containerregistry
	// client.
	signedPayloads, err := cosign.FetchSignaturesForReference(ctx, imgRef,
		ociremote.WithRemoteOptions(optionsFromRegistry(registry)...))

	// Cosign will return an error in case no signature is associated, we don't want to return that error. Since no
	// error types are exposed need to check for string comparison.
	// Cosign ref:
	//  https://github.com/sigstore/cosign/blob/44f3814667ba6a398aef62814cabc82aee4896e5/pkg/cosign/fetch.go#L84-L86
	if err != nil && !strings.Contains(err.Error(), "no signatures associated") {
		log.Errorf("Fetching signature for image %q: %v", imgFullName, err)
		return nil, false
	}

	// Short-circuit if no signatures are associated with the image.
	if len(signedPayloads) == 0 {
		return nil, false
	}

	cosignSignatures := make([]*storage.Signature, 0, len(signedPayloads))

	for _, signedPayload := range signedPayloads {
		rawSig, err := base64.StdEncoding.DecodeString(signedPayload.Base64Signature)
		// We skip the invalid base64 signature and log its occurrence.
		if err != nil {
			log.Errorf("Error during decoding of raw signature for image %q: %v",
				imgFullName, err)
		}
		// Since we are only focusing on public keys, we are ignoring the certificate / rekor bundles associated with
		// the signature.
		cosignSignatures = append(cosignSignatures, &storage.Signature{
			Signature: &storage.Signature_Cosign{
				Cosign: &storage.CosignSignature{
					RawSignature:     rawSig,
					SignaturePayload: signedPayload.Payload,
				},
			},
		})
	}

	// Since we are skipping invalid base64 signatures, need to check the length of the result.
	if len(cosignSignatures) == 0 {
		return nil, false
	}

	return &storage.ImageSignature{
		Signatures: cosignSignatures,
	}, true
}

func optionsFromRegistry(registry registryTypes.ImageRegistry) []gcrRemote.Option {
	registryCfg := &registryTypes.Config{}
	if cfg := registry.Config(); cfg != nil {
		registryCfg = cfg
	}
	authCfg := authn.AuthConfig{
		Username: registryCfg.Username,
		Password: registryCfg.Password,
	}

	auth := authn.FromConfig(authCfg)

	// By default, the proxy will be taken from environment, assuming this will be in line with our general proxy
	// strategy.
	transport := gcrRemote.DefaultTransport
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: registryCfg.Insecure}

	return []gcrRemote.Option{
		gcrRemote.WithAuth(auth),
		gcrRemote.WithTransport(transport),
	}
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
		return nil, gcrv1.Hash{}, errox.Newf(errHashCreation, err.Error())
	}

	// Theoretically, this should never happen, as gcrv1.NewHash _currently_ doesn't support any other hash algorithm.
	// See: https://github.com/google/go-containerregistry/blob/main/pkg/v1/hash.go#L78
	// We should keep this check although, in case there are changes in the library.
	if hash.Algorithm != sha256Algo {
		return nil, gcrv1.Hash{}, errox.Newf(errInvalidHashAlgo,
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
			return nil, gcrv1.Hash{}, errox.Newf(errCorruptedSignature, err.Error())
		}
		signatures = append(signatures, sig)
	}

	return signatures, hash, nil
}
