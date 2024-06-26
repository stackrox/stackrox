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
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/fulcio"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoutils"
)

const (
	publicKeyType = "PUBLIC KEY"
	sha256Algo    = "sha256"
)

var (
	errNoImageSHA         = errors.New("no image SHA found")
	errInvalidHashAlgo    = errox.InvalidArgs.New("invalid hash algorithm used")
	errNoVerificationData = errors.New("verification data not found")
	errHashCreation       = errox.InvariantViolation.New("creating hash")
	errCorruptedSignature = errox.InvariantViolation.New("corrupted signature")
)

type cosignSignatureVerifier struct {
	parsedPublicKeys []crypto.PublicKey
	certs            []certVerificationData

	verifierOpts []cosign.CheckOpts
}

type certVerificationData struct {
	cert           *x509.Certificate
	chain          []*x509.Certificate
	oidcIssuerExpr string
	identityExpr   string
}

var _ SignatureVerifier = (*cosignSignatureVerifier)(nil)

// IsValidPublicKeyPEMBlock is a helper function which checks whether public key PEM block was successfully decoded.
func IsValidPublicKeyPEMBlock(keyBlock *pem.Block, rest []byte) bool {
	return keyBlock != nil && keyBlock.Type == publicKeyType && len(rest) == 0
}

// newCosignSignatureVerifier creates a public key verifier with the given Cosign configuration. The provided public keys
// MUST be valid PEM encoded ones.
// It will return an error if the provided public keys could not be parsed.
func newCosignSignatureVerifier(config *storage.SignatureIntegration) (*cosignSignatureVerifier, error) {
	publicKeys := config.GetCosign().GetPublicKeys()
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

	cosignCerts := config.GetCosignCertificates()
	certsWithChains := make([]certVerificationData, 0, len(cosignCerts))
	for _, cosignCert := range cosignCerts {
		var cert *x509.Certificate
		if certPEM := cosignCert.GetCertificatePemEnc(); certPEM != "" {
			certs, err := cryptoutils.UnmarshalCertificatesFromPEM([]byte(certPEM))
			if err != nil {
				return nil, errox.InvariantViolation.New("failed to unmarshal certificate from PEM")
			}
			if len(certs) != 0 {
				cert = certs[0]
			}
		}

		var chain []*x509.Certificate
		if chainPEM := cosignCert.GetCertificateChainPemEnc(); chainPEM != "" {
			c, err := cryptoutils.UnmarshalCertificatesFromPEM([]byte(chainPEM))
			if err != nil {
				return nil, errox.InvariantViolation.New("failed to unmarshal certificate chain PEM")
			}
			chain = c
		}

		certsWithChains = append(certsWithChains, certVerificationData{
			chain:          chain,
			cert:           cert,
			oidcIssuerExpr: cosignCert.GetCertificateOidcIssuer(),
			identityExpr:   cosignCert.GetCertificateIdentity(),
		})
	}

	return &cosignSignatureVerifier{parsedPublicKeys: parsedKeys, certs: certsWithChains}, nil
}

// VerifySignature implements the SignatureVerifier interface.
// The signature of the image will be verified using cosign. It will include the verification via public key
// as well as the claim verification of the payload of the signature.
func (c *cosignSignatureVerifier) VerifySignature(ctx context.Context,
	image *storage.Image) (storage.ImageSignatureVerificationResult_Status, []string, error) {
	// Short-circuit if we, for some reason, do not have anything to verify against.
	if len(c.parsedPublicKeys) == 0 && len(c.certs) == 0 {
		return storage.ImageSignatureVerificationResult_FAILED_VERIFICATION, nil, errNoVerificationData
	}

	sigs, hash, err := retrieveVerificationDataFromImage(image)
	if err != nil {
		return getVerificationResultStatusFromErr(err), nil, err
	}

	var allVerifyErrs error
	if err := c.createVerifierOpts(); err != nil {
		// Fail open here instead of closed. During the creation of verifier opts for certificates, if one is given,
		// verification of the subject & identity will be done. Thus, it could always fail if things aren't signed
		// appropriately. In case we have a signature integration with a mix of keys & certificates to verify against,
		// let's first try to go through any options that might have been successfully created (i.e. the key ones)
		// and attempt to verify the signature.
		allVerifyErrs = multierror.Append(allVerifyErrs, err)
	}

	for _, opts := range c.verifierOpts {
		verifiedImageReferences, err := verifyImageSignatures(ctx, sigs, hash, image, opts)
		if err == nil && len(verifiedImageReferences) != 0 {
			return storage.ImageSignatureVerificationResult_VERIFIED, verifiedImageReferences, nil
		}
		allVerifyErrs = multierror.Append(allVerifyErrs, err)
	}

	return storage.ImageSignatureVerificationResult_FAILED_VERIFICATION, nil, allVerifyErrs
}

func (c *cosignSignatureVerifier) createVerifierOpts() error {
	var verifierErrs error

	for _, key := range c.parsedPublicKeys {
		// For now, only supporting SHA256 as algorithm.
		v, err := signature.LoadVerifier(key, crypto.SHA256)
		if err != nil {
			verifierErrs = multierror.Append(verifierErrs, errors.Wrap(err, "creating verifier"))
			continue
		}
		opts := defaultCosignCheckOpts()
		opts.SigVerifier = v
		c.verifierOpts = append(c.verifierOpts, opts)
	}

	for _, certs := range c.certs {
		opts, err := cosignCheckOptsFromCert(certs)
		if err != nil {
			verifierErrs = multierror.Append(verifierErrs, errors.Wrap(err, "creating cosign check opts"))
			continue
		}
		c.verifierOpts = append(c.verifierOpts, opts)
	}

	return verifierErrs
}

func defaultCosignCheckOpts() cosign.CheckOpts {
	return cosign.CheckOpts{
		ClaimVerifier: cosign.SimpleClaimVerifier,
		// With the latest version of cosign, by default signatures will be uploaded to rekor.
		// This means that also during verification, an entry in the transparency log will be expected and verified.
		// Since currently this is not the case in what we support / offer, explicitly disable this for now
		// until we enable and expect this to be the case.
		IgnoreSCT:  true,
		IgnoreTlog: true,
	}
}

func cosignCheckOptsFromCert(cert certVerificationData) (cosign.CheckOpts, error) {
	opts := defaultCosignCheckOpts()

	// Skip verifying the identities when the wildcard matching logic being used. This fixes an issue
	// with verifying the identity which will yield an error when using the wildcard expressions _and_ the certificate
	// to verify has no identities associated with it (i.e. within the BYOPKI use-case).
	if cert.oidcIssuerExpr != ".*" && cert.identityExpr != ".*" {
		opts.Identities = []cosign.Identity{{
			IssuerRegExp:  cert.oidcIssuerExpr,
			SubjectRegExp: cert.identityExpr,
		}}
	}

	var err error
	// - If we have both cert and chain, we use both to verify the public key and the root.
	// - If we only have the cert, we assume the fulcio trusted root.
	// - If we only have the chain, we use this as the trusted root to verify certificates, if any.
	// - If none is given, we set the fulcio roots.
	switch {
	case cert.cert != nil && len(cert.chain) > 0:
		v, err := cosign.ValidateAndUnpackCertWithChain(cert.cert, cert.chain, &opts)
		if err != nil {
			return opts, err
		}
		opts.SigVerifier = v
		return opts, nil

	case cert.cert != nil:
		opts.RootCerts, err = fulcio.GetRoots()
		if err != nil {
			return opts, err
		}
		opts.IntermediateCerts, err = fulcio.GetIntermediates()
		if err != nil {
			return opts, err
		}
		v, err := cosign.ValidateAndUnpackCert(cert.cert, &opts)
		if err != nil {
			return opts, err
		}
		opts.SigVerifier = v
		return opts, nil

	case len(cert.chain) > 0:
		pool := x509.NewCertPool()
		for _, caCert := range cert.chain {
			pool.AddCert(caCert)
		}
		opts.RootCerts = pool
		return opts, nil

	default:
		opts.RootCerts, err = fulcio.GetRoots()
		if err != nil {
			return opts, err
		}
		return opts, nil
	}
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

		sig, err := static.NewSignature(imgSig.GetCosign().GetSignaturePayload(), b64Sig,
			static.WithCertChain(imgSig.GetCosign().GetCertPem(), imgSig.GetCosign().GetCertChainPem()))
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
	imageNames := protoutils.SliceUnique(append(image.GetNames(), image.GetName()))
	for _, name := range imageNames {
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
