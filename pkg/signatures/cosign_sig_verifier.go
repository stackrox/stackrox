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
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	"github.com/sigstore/cosign/v3/pkg/cosign/bundle"
	"github.com/sigstore/cosign/v3/pkg/oci"
	"github.com/sigstore/cosign/v3/pkg/oci/static"
	rekorClient "github.com/sigstore/rekor/pkg/client"
	sgbundle "github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
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

// verifiableSignature holds signature data ready for verification. Exactly one of
// the two fields is populated:
//   - sigstoreBundle: raw sigstore bundle JSON, verified via verifySigstoreBundle (sigstore-go).
//   - signature: reconstructed oci.Signature from decomposed SimpleSigning fields, verified via
//     cosign.VerifyImageSignature.
type verifiableSignature struct {
	sigstoreBundle []byte
	signature      oci.Signature
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
				verifierRefs, err := verifySignature(ctx, vsig.sigstoreBundle, vsig.signature, hash, image, opts)
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

// transparencyLogsFromKeys converts cosign's TrustedTransparencyLogPubKeys map into
// sigstore-go TransparencyLog entries.
func transparencyLogsFromKeys(keys *cosign.TrustedTransparencyLogPubKeys) map[string]*root.TransparencyLog {
	if keys == nil || len(keys.Keys) == 0 {
		return nil
	}
	logs := make(map[string]*root.TransparencyLog, len(keys.Keys))
	for logID, key := range keys.Keys {
		idBytes, err := hex.DecodeString(logID)
		if err != nil {
			log.Warnf("Skipping transparency log key with malformed log ID %q: %v", logID, err)
			continue
		}
		logs[logID] = &root.TransparencyLog{
			ID:                idBytes,
			PublicKey:         key.PubKey,
			HashFunc:          crypto.SHA256,
			SignatureHashFunc: crypto.SHA256,
			// sigstore-go VerifySET rejects zero-value ValidityPeriodStart
			// as "not set". Unix epoch is a non-zero sentinel meaning "always valid".
			ValidityPeriodStart: time.Unix(0, 0),
		}
	}
	return logs
}

// trustedMaterialFromTlogKeys builds a root.TrustedMaterial from the Rekor and CTLog keys
// on CheckOpts. Returns nil if no custom keys are configured (verifySigstoreBundle will
// fall back to the public Sigstore TUF root).
func trustedMaterialFromTlogKeys(opts cosign.CheckOpts) (root.TrustedMaterial, error) {
	rekorLogs := transparencyLogsFromKeys(opts.RekorPubKeys)
	ctLogs := transparencyLogsFromKeys(opts.CTLogPubKeys)
	if len(rekorLogs) == 0 && len(ctLogs) == 0 {
		return nil, nil
	}
	tr, err := root.NewTrustedRoot(root.TrustedRootMediaType01, nil, ctLogs, nil, rekorLogs)
	if err != nil {
		return nil, fmt.Errorf("building trusted root from transparency log keys: %w", err)
	}
	return tr, nil
}

// trustedMaterialFromCertificateChain builds a root.TrustedMaterial from a custom certificate
// chain. The last cert in the chain is treated as the root; the rest are intermediates.
func trustedMaterialFromCertificateChain(chain []*x509.Certificate, opts cosign.CheckOpts) (root.TrustedMaterial, error) {
	ca := &root.FulcioCertificateAuthority{
		Root:          chain[len(chain)-1],
		Intermediates: chain[:len(chain)-1],
	}
	return root.NewTrustedRoot(
		root.TrustedRootMediaType01,
		[]root.CertificateAuthority{ca},
		transparencyLogsFromKeys(opts.CTLogPubKeys),
		nil,
		transparencyLogsFromKeys(opts.RekorPubKeys),
	)
}

// augmentTrustedMaterialWithSigChain extracts intermediates from a SimpleSigning
// signature's embedded certificate chain and merges them into the TrustedMaterial.
// This is necessary because cosign's ValidateAndUnpackCertWithIntermediates ignores
// sig.Chain() intermediates when TrustedMaterial is set, but the configured chain
// may only contain the root CA (BYOPKI pattern where the signer provides the
// intermediate chain).
func augmentTrustedMaterialWithSigChain(cosignOpts *cosign.CheckOpts, sig oci.Signature) error {
	sigChain, err := sig.Chain()
	if err != nil || len(sigChain) <= 1 {
		return err
	}
	sigIntermediates := sigChain[:len(sigChain)-1]

	cas := cosignOpts.TrustedMaterial.FulcioCertificateAuthorities()
	augmented := make([]root.CertificateAuthority, 0, len(cas))
	for _, ca := range cas {
		fca, ok := ca.(*root.FulcioCertificateAuthority)
		if !ok {
			augmented = append(augmented, ca)
			continue
		}
		merged := &root.FulcioCertificateAuthority{
			Root:                fca.Root,
			Intermediates:       append(append([]*x509.Certificate{}, fca.Intermediates...), sigIntermediates...),
			ValidityPeriodStart: fca.ValidityPeriodStart,
			ValidityPeriodEnd:   fca.ValidityPeriodEnd,
		}
		augmented = append(augmented, merged)
	}

	tm, err := root.NewTrustedRoot(
		root.TrustedRootMediaType01,
		augmented,
		cosignOpts.TrustedMaterial.CTLogs(),
		cosignOpts.TrustedMaterial.TimestampingAuthorities(),
		cosignOpts.TrustedMaterial.RekorLogs(),
	)
	if err != nil {
		return fmt.Errorf("rebuilding trusted material with signature intermediates: %w", err)
	}
	cosignOpts.TrustedMaterial = tm
	return nil
}

// sigstoreTrustedRoot returns the public Sigstore trusted root from TUF.
func sigstoreTrustedRoot() (root.TrustedMaterial, error) {
	tr, err := cosign.TrustedRoot()
	if err != nil {
		return nil, fmt.Errorf("loading sigstore trusted root: %w", err)
	}
	return tr, nil
}

func (c *cosignSignatureVerifier) createVerifierOpts(ctx context.Context) error {
	defaultOpts, err := c.defaultCosignCheckOpts(ctx)
	if err != nil {
		return errors.Wrap(err, "creating default cosign check opts")
	}

	// Build TrustedMaterial once from the tlog keys for public key verifiers.
	publicKeyTrustedMaterial, err := trustedMaterialFromTlogKeys(defaultOpts)
	if err != nil {
		return err
	}

	var verifierErrs error
	// Public key verifiers.
	for _, key := range c.parsedPublicKeys {
		v, err := signature.LoadVerifier(key, crypto.SHA256)
		if err != nil {
			verifierErrs = multierror.Append(verifierErrs, errors.Wrap(err, "creating verifier"))
			continue
		}
		opts := defaultOpts
		opts.SigVerifier = v
		opts.TrustedMaterial = publicKeyTrustedMaterial
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
	// Only set non-wildcard identities. CheckCertificatePolicy fails on BYOPKI certs
	// that lack OIDC extensions. For the sigstore-go bundle path, verifySigstoreBundle
	// injects wildcard identities when Identities is empty and SigVerifier is nil.
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

	// Build TrustedMaterial: custom chain when provided, Sigstore (Fulcio) root otherwise.
	// All certificate chain validation flows through TrustedMaterial — the legacy
	// RootCerts/IntermediateCerts fields are not needed.
	if len(cert.chain) > 0 {
		opts.TrustedMaterial, err = trustedMaterialFromCertificateChain(cert.chain, opts)
		if err != nil {
			return opts, err
		}
	} else {
		opts.TrustedMaterial, err = sigstoreTrustedRoot()
		if err != nil {
			return opts, err
		}
	}

	// When a leaf certificate is provided, validate it against TrustedMaterial and
	// pin its public key as SigVerifier. This catches misconfigurations early (cert
	// doesn't chain to the configured/Fulcio root, missing SCT) and pins the exact
	// signing key so only signatures from this key are accepted.
	if cert.cert != nil {
		chains, err := verify.VerifyLeafCertificate(time.Now(), cert.cert, opts.TrustedMaterial)
		if err != nil {
			return opts, fmt.Errorf("validating configured certificate against trust root: %w", err)
		}
		if err := cosign.CheckCertificatePolicy(cert.cert, &opts); err != nil {
			return opts, err
		}
		if !opts.IgnoreSCT {
			contains, err := cosign.ContainsSCT(cert.cert.Raw)
			if err != nil {
				return opts, err
			}
			if !contains {
				return opts, errors.New("certificate does not include required embedded SCT")
			}
			if err := verify.VerifySignedCertificateTimestamp(chains, 1, opts.TrustedMaterial); err != nil {
				return opts, fmt.Errorf("verifying signed certificate timestamp: %w", err)
			}
		}
		v, err := signature.LoadVerifier(cert.cert.PublicKey, crypto.SHA256)
		if err != nil {
			return opts, errors.Wrap(err, "loading verifier from certificate")
		}
		opts.SigVerifier = v
	}

	return opts, nil
}

// verifySignature dispatches signature verification based on format:
//   - Sigstore bundle: delegates to verifySigstoreBundle which uses cosign.VerifyNewBundle
//     (sigstore-go). The bundle carries its own verification material (cert, tlog entry).
//   - SimpleSigning: delegates to cosign.VerifyImageSignature which verifies the signature
//     and rekor bundle (transparency log proof) from the decomposed proto fields.
func verifySignature(ctx context.Context, sigstoreBundle []byte, sig oci.Signature,
	imageHash gcrv1.Hash, image *storage.Image, cosignOpts cosign.CheckOpts,
) ([]string, error) {
	// Verify DSSE Sigstore bundle.
	if len(sigstoreBundle) > 0 {
		return verifySigstoreBundle(ctx, sigstoreBundle, imageHash, image, cosignOpts)
	}

	// Verify SimpleSigning signature.
	// cosign's ValidateAndUnpackCertWithIntermediates uses VerifyLeafCertificate when
	// TrustedMaterial is set, which only considers CAs from the TrustedMaterial and
	// ignores intermediates embedded in the signature. Merge them in so the chain
	// builds correctly for BYOPKI setups where the configured chain is just the root.
	if cosignOpts.TrustedMaterial != nil {
		if err := augmentTrustedMaterialWithSigChain(&cosignOpts, sig); err != nil {
			return nil, err
		}
	}
	rekorVerified, err := cosign.VerifyImageSignature(ctx, sig, imageHash, &cosignOpts)
	if err != nil {
		return nil, err
	}
	if !rekorVerified && !cosignOpts.IgnoreTlog {
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

// verifySigstoreBundle verifies a sigstore bundle directly via cosign.VerifyNewBundle.
// TrustedMaterial must be set by the caller (createVerifierOpts / cosignCheckOptsFromCert).
func verifySigstoreBundle(ctx context.Context, sigstoreBundle []byte,
	imageHash gcrv1.Hash, image *storage.Image, cosignOpts cosign.CheckOpts,
) ([]string, error) {
	if cosignOpts.TrustedMaterial == nil {
		var err error
		if cosignOpts.IgnoreTlog {
			cosignOpts.TrustedMaterial, err = root.NewTrustedRoot(root.TrustedRootMediaType01, nil, nil, nil, nil)
		} else {
			cosignOpts.TrustedMaterial, err = sigstoreTrustedRoot()
		}
		if err != nil {
			return nil, fmt.Errorf("loading trusted material for bundle verification: %w", err)
		}
	}

	// sigstore-go requires at least one identity for cert-based verification.
	// cosignCheckOptsFromCert skips Identities for wildcard BYOPKI certs (they lack
	// OIDC extensions that the legacy path would check), so inject them here.
	if len(cosignOpts.Identities) == 0 && cosignOpts.SigVerifier == nil {
		cosignOpts.Identities = []cosign.Identity{{SubjectRegExp: ".*", IssuerRegExp: ".*"}}
	}

	bundle := &sgbundle.Bundle{}
	if err := bundle.UnmarshalJSON(sigstoreBundle); err != nil {
		return nil, fmt.Errorf("unmarshalling sigstore bundle: %w", err)
	}

	digestBytes, err := hex.DecodeString(imageHash.Hex)
	if err != nil {
		return nil, fmt.Errorf("decoding image digest hex: %w", err)
	}

	artifactPolicy := verify.WithArtifactDigest(imageHash.Algorithm, digestBytes)
	if _, err := cosign.VerifyNewBundle(ctx, &cosignOpts, artifactPolicy, bundle); err != nil {
		return nil, err
	}

	// Sigstore bundles bind to the image digest, not a docker reference, so all
	// names sharing the same digest are considered verified.
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

// retrieveVerificationDataFromImage reads stored signatures from the image proto and
// prepares them for verification. Two storage formats are handled:
//   - SigstoreBundle set: the raw bundle JSON is carried through for verifySigstoreBundle.
//   - SigstoreBundle empty (legacy): the decomposed fields (RawSignature, SignaturePayload,
//     CertPem, CertChainPem, RekorBundle) are reassembled into an oci.Signature for
//     cosign.VerifyImageSignature.
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
				sigstoreBundle: cosignSig.GetSigstoreBundle(),
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
			signature: sig,
		})
	}

	return vsigs, hash, nil
}

// getVerifiedImageReferenceFromSignature retrieves the verified docker reference in the format of
// <registry>/<repository> from the payload of the SimpleSigning oci.Signature and filters out image
// names that are verified by the docker reference using the image names associated with the storage.Image.
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
	signatureIdentity := simpleContainer.Critical.Identity.DockerReference
	log.Debugf("Retrieving verified image references from the image names [%v] and signature identity %q",
		image.GetNames(), signatureIdentity)
	var verifiedImageReferences []string
	// We must ensure here that `append` is not called directly on the result of
	// `image.GetNames()`. Otherwise, we create a data race caused by concurrent
	// writes to the underlying data array of the slice.
	imageNames := protoutils.SliceUnique(
		append([]*storage.ImageName{image.GetName()}, image.GetNames()...),
	)
	for _, name := range imageNames {
		ok, err := equalRegistryRepository(signatureIdentity, name.GetFullName())
		if err != nil {
			// Theoretically, all references should be parsable.
			// In case we somehow get an invalid entry, we will log the occurrence and skip this entry.
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
