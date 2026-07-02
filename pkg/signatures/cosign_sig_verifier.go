package signatures

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/fulcio"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	"github.com/sigstore/cosign/v3/pkg/cosign/bundle"
	"github.com/sigstore/cosign/v3/pkg/oci"
	"github.com/sigstore/cosign/v3/pkg/oci/static"
	rekorClient "github.com/sigstore/rekor/pkg/client"
	sgbundle "github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/sigstore/sigstore/pkg/tuf"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	publicKeyType = "PUBLIC KEY"
	sha256Algo    = "sha256"
)

var (
	errCorruptedSignature   = errox.InvariantViolation.New("corrupted signature")
	errHashCreation         = errox.InvariantViolation.New("creating hash")
	errInvalidHashAlgo      = errox.InvalidArgs.New("invalid hash algorithm used")
	errNoImageSHA           = errors.New("no image SHA found")
	errNoVerificationData   = errors.New("verification data not found")
	errNoVerifiedReferences = errors.New("no verified references")
	errUnverifiedBundle     = errors.New("unverified transparency log bundle")
)

type verifiableSignature struct {
	sig       oci.Signature
	rawBundle []byte
}

var once sync.Once

func setupTufRootDir() {
	once.Do(func() {
		// TufRoot sets the location of where to store the TUF roots.
		// When using Fulcio roots to verify signatures, the roots will be persisted within a temporary directory.
		// When running Central within a container, we are in a read-only file system. Set the path here explicitly
		// to a writeable directory. Unfortunately this has to be done via environment variable, since no option
		// is exposed on the TUF library to set this otherwise.
		utils.Should(os.Setenv(tuf.TufRootEnv, "/tmp/tuf-roots"))
	})
}

type cosignSignatureVerifier struct {
	parsedPublicKeys []crypto.PublicKey
	certs            []certVerificationData
	transparencyLog  *tlogVerificationData

	verifierOpts []cosign.CheckOpts
}

type certVerificationData struct {
	cert           *x509.Certificate
	chain          []*x509.Certificate
	oidcIssuerExpr string
	identityExpr   string
	ctlogEnabled   bool
	ctlogPublicKey string
}

type tlogVerificationData struct {
	enabled         bool
	publicKey       string
	url             string
	validateOffline bool
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
	setupTufRootDir()

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
			ctlogEnabled:   cosignCert.GetCertificateTransparencyLog().GetEnabled(),
			ctlogPublicKey: cosignCert.GetCertificateTransparencyLog().GetPublicKeyPemEnc(),
		})
	}

	tlog := config.GetTransparencyLog()
	tlogVerificationData := &tlogVerificationData{
		enabled:         tlog.GetEnabled(),
		url:             tlog.GetUrl(),
		validateOffline: tlog.GetValidateOffline(),
		publicKey:       tlog.GetPublicKeyPemEnc(),
	}

	return &cosignSignatureVerifier{
		parsedPublicKeys: parsedKeys,
		certs:            certsWithChains,
		transparencyLog:  tlogVerificationData,
	}, nil
}

// VerifySignature implements the SignatureVerifier interface.
// The signature of the image will be verified using cosign. It will include the verification via public key
// as well as the claim verification of the payload of the signature.
func (c *cosignSignatureVerifier) VerifySignature(ctx context.Context,
	image *storage.Image,
) (storage.ImageSignatureVerificationResult_Status, []string, error) {
	// Short-circuit if we, for some reason, do not have anything to verify against.
	if len(c.parsedPublicKeys) == 0 && len(c.certs) == 0 {
		return storage.ImageSignatureVerificationResult_FAILED_VERIFICATION, nil, errNoVerificationData
	}

	vsigs, hash, err := retrieveVerificationDataFromImage(image)
	if err != nil {
		return getVerificationResultStatusFromErr(err), nil, err
	}

	var allVerifyErrs error
	if err := c.createVerifierOpts(ctx); err != nil {
		// Fail open here instead of closed. During the creation of verifier opts for certificates, if one is given,
		// verification of the subject & identity will be done. Thus, it could always fail if things aren't signed
		// appropriately. In case we have a signature integration with a mix of keys & certificates to verify against,
		// let's first try to go through any options that might have been successfully created (i.e. the key ones)
		// and attempt to verify the signature.
		allVerifyErrs = multierror.Append(allVerifyErrs, err)
	}

	// Find the union of all image references from verified signatures. The resulting status
	// is verified if at least one verification was successful.
	//
	// verifier_1(sig_1) OR ... OR verifier_1(sig_N)
	// OR
	// ...
	// OR
	// verifier_N(sig_1) OR ... OR verifier_N(sig_N)
	verifiedImageReferences := set.NewStringSet()
	var mutex sync.Mutex
	var wg sync.WaitGroup
	for _, opts := range c.verifierOpts {
		for cnt, vsig := range vsigs {
			wg.Go(func() {
				verifierRefs, err := verifyImageSignature(ctx, vsig.sig, hash, image, opts, vsig.rawBundle)
				mutex.Lock()
				defer mutex.Unlock()
				if err != nil {
					allVerifyErrs = multierror.Append(
						allVerifyErrs,
						errors.Wrapf(err, "verifying signature %d", cnt+1),
					)
					return
				}
				verifiedImageReferences.AddAll(verifierRefs...)
			})
		}
	}
	wg.Wait()

	if len(verifiedImageReferences) > 0 {
		verifiedRefSlice := verifiedImageReferences.AsSortedSlice(
			func(i, j string) bool { return i < j },
		)
		return storage.ImageSignatureVerificationResult_VERIFIED, verifiedRefSlice, nil
	}
	return storage.ImageSignatureVerificationResult_FAILED_VERIFICATION, nil, allVerifyErrs
}

func (c *cosignSignatureVerifier) createVerifierOpts(ctx context.Context) error {
	defaultOpts, err := c.defaultCosignCheckOpts(ctx)
	if err != nil {
		return errors.Wrap(err, "creating default cosign check opts")
	}

	var verifierErrs error
	// Public key verifiers.
	for _, key := range c.parsedPublicKeys {
		// For now, only supporting SHA256 as algorithm.
		v, err := signature.LoadVerifier(key, crypto.SHA256)
		if err != nil {
			verifierErrs = multierror.Append(verifierErrs, errors.Wrap(err, "creating verifier"))
			continue
		}
		opts := defaultOpts
		opts.SigVerifier = v
		c.verifierOpts = append(c.verifierOpts, opts)
	}

	// Certificate verifiers.
	for _, cert := range c.certs {
		opts, err := cosignCheckOptsFromCert(ctx, cert, defaultOpts)
		if err != nil {
			verifierErrs = multierror.Append(verifierErrs, errors.Wrap(err, "creating cosign check opts from cert"))
			continue
		}
		c.verifierOpts = append(c.verifierOpts, opts)
	}

	return verifierErrs
}

func newTrustedTransparencyLogPubKeys(publicKey string) (*cosign.TrustedTransparencyLogPubKeys, error) {
	publicKeys := cosign.NewTrustedTransparencyLogPubKeys()
	if err := publicKeys.AddTransparencyLogPubKey([]byte(publicKey), tuf.Active); err != nil {
		return nil, err
	}
	return &publicKeys, nil
}

func getCTLogPublicKeys(ctx context.Context, publicKey string) (*cosign.TrustedTransparencyLogPubKeys, error) {
	if publicKey == "" {
		return cosign.GetCTLogPubs(ctx)
	}
	return newTrustedTransparencyLogPubKeys(publicKey)
}

func getRekorPublicKeys(ctx context.Context, publicKey string) (*cosign.TrustedTransparencyLogPubKeys, error) {
	if publicKey == "" {
		return cosign.GetRekorPubs(ctx)
	}
	return newTrustedTransparencyLogPubKeys(publicKey)
}

func (c *cosignSignatureVerifier) setDefaultTlogCheckOpts(ctx context.Context, opts *cosign.CheckOpts) error {
	opts.IgnoreTlog = !c.transparencyLog.enabled
	if opts.IgnoreTlog {
		return nil
	}

	var err error
	opts.RekorPubKeys, err = getRekorPublicKeys(ctx, c.transparencyLog.publicKey)
	if err != nil {
		return errors.Wrap(err, "getting rekor public keys")
	}

	opts.Offline = c.transparencyLog.validateOffline
	if opts.Offline {
		return nil
	}

	formattedURL := urlfmt.FormatURL(c.transparencyLog.url, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	opts.RekorClient, err = rekorClient.GetRekorClient(formattedURL)
	if err != nil {
		return errors.Wrap(err, "creating rekor client")
	}
	return nil
}

func (c *cosignSignatureVerifier) defaultCosignCheckOpts(ctx context.Context) (cosign.CheckOpts, error) {
	opts := cosign.CheckOpts{ClaimVerifier: cosign.SimpleClaimVerifier}
	if err := c.setDefaultTlogCheckOpts(ctx, &opts); err != nil {
		return cosign.CheckOpts{}, err
	}
	return opts, nil
}

func cosignCheckOptsFromCert(ctx context.Context, cert certVerificationData, opts cosign.CheckOpts) (cosign.CheckOpts, error) {
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
	opts.IgnoreSCT = !cert.ctlogEnabled
	if !opts.IgnoreSCT {
		opts.CTLogPubKeys, err = getCTLogPublicKeys(ctx, cert.ctlogPublicKey)
		if err != nil {
			return opts, errors.Wrap(err, "getting ctlog public keys")
		}
	}

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

// verifyImageSignature verifies a cosign image signature. Sigstore bundles are verified
// via cosign.VerifyNewBundle (sigstore-go). Legacy SimpleSigning signatures use
// cosign.VerifyImageSignature.
func verifyImageSignature(ctx context.Context, sig oci.Signature,
	imageHash gcrv1.Hash, image *storage.Image, cosignOpts cosign.CheckOpts,
	rawBundle []byte,
) ([]string, error) {
	if len(rawBundle) > 0 {
		return verifyBundleSignature(ctx, imageHash, image, cosignOpts, rawBundle)
	}

	bundleVerified, err := cosign.VerifyImageSignature(ctx, sig, imageHash, &cosignOpts)
	if err != nil {
		return nil, err
	}
	if !bundleVerified && !cosignOpts.IgnoreTlog {
		return nil, errUnverifiedBundle
	}

	refs, err := getVerifiedImageReference(sig, image)
	if err != nil {
		return nil, errors.Wrap(err, "getting verified image references")
	}
	if len(refs) == 0 {
		return nil, errNoVerifiedReferences
	}
	return refs, nil
}

// verifyBundleSignature verifies a sigstore bundle directly via cosign.VerifyNewBundle.
// TrustedMaterial is loaded lazily here (not in defaultCosignCheckOpts) because it is
// exclusive with the legacy RootCerts/IntermediateCerts fields used by non-bundle paths.
func verifyBundleSignature(ctx context.Context,
	imageHash gcrv1.Hash, image *storage.Image, cosignOpts cosign.CheckOpts,
	rawBundle []byte,
) ([]string, error) {
	tr, err := cosign.TrustedRoot()
	if err != nil {
		return nil, fmt.Errorf("loading sigstore trusted root for bundle verification: %w", err)
	}
	cosignOpts.TrustedMaterial = tr

	b := &sgbundle.Bundle{}
	if err := b.UnmarshalJSON(rawBundle); err != nil {
		return nil, fmt.Errorf("unmarshalling sigstore bundle: %w", err)
	}

	digestBytes, err := hex.DecodeString(imageHash.Hex)
	if err != nil {
		return nil, fmt.Errorf("decoding image digest hex: %w", err)
	}

	artifactPolicy := verify.WithArtifactDigest(imageHash.Algorithm, digestBytes)
	if _, err := cosign.VerifyNewBundle(ctx, &cosignOpts, artifactPolicy, b); err != nil {
		return nil, err
	}

	return getAllImageReferences(image), nil
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

func unmarshalRekorBundle(byteBundle []byte) (*bundle.RekorBundle, error) {
	// Nil pointers are marshalled to "null" instead of empty slices.
	if len(byteBundle) == 0 || string(byteBundle) == "null" {
		return nil, nil
	}
	// Need to force string type for RekorBundle.Payload.Body because it is defined as
	// an untyped interface. The Unmarshal is type-confused otherwise.
	rekorBundle := &bundle.RekorBundle{Payload: bundle.RekorPayload{Body: ""}}
	err := json.Unmarshal(byteBundle, rekorBundle)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling rekor bundle")
	}
	return rekorBundle, nil
}

func retrieveVerificationDataFromImage(image *storage.Image) ([]verifiableSignature, gcrv1.Hash, error) {
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

	vsigs := make([]verifiableSignature, 0, len(image.GetSignature().GetSignatures()))
	for _, imgSig := range image.GetSignature().GetSignatures() {
		if imgSig.GetCosign() == nil {
			continue
		}

		cosignSig := imgSig.GetCosign()

		// Sigstore bundle: carry the raw bundle for direct sigstore-go verification.
		if len(cosignSig.GetSigstoreBundle()) > 0 {
			vsigs = append(vsigs, verifiableSignature{
				rawBundle: cosignSig.GetSigstoreBundle(),
			})
			continue
		}

		// Legacy decomposed fields.
		b64Sig := base64.StdEncoding.EncodeToString(cosignSig.GetRawSignature())
		sigOpts := []static.Option{
			static.WithCertChain(cosignSig.GetCertPem(), cosignSig.GetCertChainPem()),
		}

		rekorBundle, err := unmarshalRekorBundle(cosignSig.GetRekorBundle())
		if err != nil {
			log.Errorf("Failed to unmarshal rekor bundle for image %q: %s", image.GetName().GetFullName(), err)
		}
		if rekorBundle != nil {
			sigOpts = append(sigOpts, static.WithBundle(rekorBundle))
		}

		sig, err := static.NewSignature(cosignSig.GetSignaturePayload(), b64Sig, sigOpts...)
		if err != nil {
			return nil, gcrv1.Hash{}, errCorruptedSignature.CausedBy(err)
		}
		vsigs = append(vsigs, verifiableSignature{
			sig: sig,
		})
	}

	return vsigs, hash, nil
}

// getVerifiedImageReference returns image names verified by a SimpleSigning signature.
// The payload carries a docker reference; only names matching registry+repository are returned.
func getVerifiedImageReference(signature oci.Signature, image *storage.Image) ([]string, error) {
	payloadBytes, err := signature.Payload()
	if err != nil {
		return nil, err
	}
	var simpleContainer payload.SimpleContainerImage
	if err := json.Unmarshal(payloadBytes, &simpleContainer); err != nil {
		return nil, err
	}

	signatureIdentity := simpleContainer.Critical.Identity.DockerReference
	log.Debugf("Retrieving verified image references from the image names [%v] and signature identity %q",
		image.GetNames(), signatureIdentity)
	var verifiedImageReferences []string
	imageNames := protoutils.SliceUnique(
		append([]*storage.ImageName{image.GetName()}, image.GetNames()...),
	)
	for _, name := range imageNames {
		ok, err := equalRegistryRepository(signatureIdentity, name.GetFullName())
		if err != nil {
			log.Errorf("Failed to compare image name %q and signature identity %q: %v", name.GetFullName(), signatureIdentity, err)
			continue
		}
		if ok {
			verifiedImageReferences = append(verifiedImageReferences, name.GetFullName())
		}
	}
	return verifiedImageReferences, nil
}

func getAllImageReferences(image *storage.Image) []string {
	imageNames := protoutils.SliceUnique(
		append([]*storage.ImageName{image.GetName()}, image.GetNames()...),
	)
	refs := make([]string, 0, len(imageNames))
	for _, n := range imageNames {
		if fullName := n.GetFullName(); fullName != "" {
			refs = append(refs, fullName)
		}
	}
	return refs
}

func equalRegistryRepository(signatureIdentity, imageName string) (bool, error) {
	sigRef, err := name.ParseReference(signatureIdentity)
	if err != nil {
		return false, errors.Wrapf(err, "parsing reference for %q", signatureIdentity)
	}
	imgRef, err := name.ParseReference(imageName)
	if err != nil {
		return false, errors.Wrapf(err, "parsing reference for %q", imageName)
	}
	return sigRef.Context().RegistryStr() == imgRef.Context().RegistryStr() &&
		sigRef.Context().RepositoryStr() == imgRef.Context().RepositoryStr(), nil
}
