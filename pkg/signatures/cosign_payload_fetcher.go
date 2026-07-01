package signatures

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	"github.com/sigstore/cosign/v3/pkg/oci"
	ociremote "github.com/sigstore/cosign/v3/pkg/oci/remote"
	"github.com/stackrox/rox/generated/storage"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
)

const (
	cosignSigArtifactType       = "application/vnd.dev.cosign.artifact.sig.v1+json"
	bundleSigArtifactTypePrefix = "application/vnd.dev.sigstore.bundle"

	// maxReferrerManifests limits the number of referrer signature manifests processed per image
	// to bound latency and prevent pathological cases from consuming the caller's timeout budget.
	maxReferrerManifests = 50
)

// signaturePayload extends cosign.SignedPayload with the signature format.
type signaturePayload struct {
	cosign.SignedPayload
	signatureFormat storage.CosignSignature_SignatureFormat
	rawBundle       []byte
}

var (
	_ oci.SignedEntity = (*tagSignedEntity)(nil)
	_ oci.SignedEntity = (*referrerSignedEntity)(nil)
)

// tagSignedEntity adapts an image reference for tag-based signature discovery.
// Signatures() constructs the cosign tag reference (<algo>-<hex>.sig) and fetches
// the signature manifest from the registry. Digest() returns the image digest so
// cosign can build the tag. Only these two methods are called by cosign.FetchSignatures.
// Attestations and Attachment return safe defaults to prevent panics if cosign evolves.
type tagSignedEntity struct {
	opts   []ociremote.Option
	imgRef name.Reference
	imgSHA string
}

func newTagSignedEntity(img *storage.Image, imgRef name.Reference, opts ...ociremote.Option) *tagSignedEntity {
	return &tagSignedEntity{
		opts:   opts,
		imgRef: imgRef,
		imgSHA: imgUtils.GetSHA(img),
	}
}

func (s *tagSignedEntity) Digest() (v1.Hash, error) {
	return v1.NewHash(s.imgSHA)
}

func (s *tagSignedEntity) Signatures() (oci.Signatures, error) {
	h, err := s.Digest()
	if err != nil {
		return nil, err
	}
	// Cosign ref: https://github.com/sigstore/cosign/blob/main/pkg/oci/remote/remote.go
	return ociremote.Signatures(s.imgRef.Context().Tag(fmt.Sprint(h.Algorithm, "-", h.Hex, ".sig")), s.opts...)
}

func (s *tagSignedEntity) Attestations() (oci.Signatures, error) { return nil, nil }

func (s *tagSignedEntity) Attachment(_ string) (oci.File, error) {
	return nil, fmt.Errorf("attachments not supported on tag-based signature entity")
}

// referrerSignedEntity adapts pre-fetched referrer signatures for cosign.FetchSignatures.
// Signatures() returns the already-fetched data. Only this method is called by
// cosign.FetchSignatures. Other methods return safe defaults.
type referrerSignedEntity struct {
	sigs oci.Signatures
}

func (r *referrerSignedEntity) Digest() (v1.Hash, error) { return v1.Hash{}, nil }

func (r *referrerSignedEntity) Signatures() (oci.Signatures, error) {
	return r.sigs, nil
}

func (r *referrerSignedEntity) Attestations() (oci.Signatures, error) { return nil, nil }

func (r *referrerSignedEntity) Attachment(_ string) (oci.File, error) {
	return nil, fmt.Errorf("attachments not supported on referrer-based signature entity")
}

// fetchTagPayloads discovers signatures via the legacy cosign tag-based method.
// It uses a local signed entity to skip fetching the image manifest and only fetch the signature manifest.
// When no signature is associated with the image, cosign returns an error which is converted to nil
// because "no signatures" is a valid state. Since cosign does not expose typed errors, we use string
// comparison (see isMissingSignatureError).
// Cosign ref: https://github.com/sigstore/cosign/blob/main/pkg/cosign/fetch.go
func fetchTagPayloads(image *storage.Image, imgRef name.Reference, opts []ociremote.Option) ([]signaturePayload, error) {
	se := newTagSignedEntity(image, imgRef, opts...)
	payloads, err := cosign.FetchSignatures(se)
	if err != nil && (isMissingSignatureError(err) || isUnknownMimeTypeError(err)) {
		return nil, nil
	}
	return wrapPayloads(payloads), err
}

func wrapPayloads(payloads []cosign.SignedPayload) []signaturePayload {
	wrapped := make([]signaturePayload, len(payloads))
	for i, p := range payloads {
		wrapped[i] = signaturePayload{SignedPayload: p}
	}
	return wrapped
}

// fetchReferrerPayloads queries the OCI 1.1 Referrers API for cosign signature artifacts
// attached to the given image digest and returns them as cosign signed payloads.
// Referrers are fetched without a server-side artifact type filter for compatibility with
// registries that do not populate the artifactType field. Client-side filtering uses a
// prefix match on the sigstore bundle media type (forward-compatible with future bundle
// versions) and an exact match on the legacy cosign signature artifact type.
// Bundle-format referrers are preferred and processed first.
func fetchReferrerPayloads(ctx context.Context, digestRef name.Digest, repo name.Repository,
	opts []ociremote.Option,
) ([]signaturePayload, error) {
	log.Infof("Fetching OCI referrer signatures for %s", digestRef.String())

	index, err := ociremote.Referrers(digestRef, "", opts...)
	if err != nil {
		// A 404 means the registry does not support the OCI 1.1 referrers endpoint.
		// The heroku ErrorTransport converts 404 responses into Go errors before
		// go-containerregistry's built-in fallback can handle them, so we catch it here.
		if checkIfErrorContainsCode(err, http.StatusNotFound) {
			log.Infof("OCI referrers API not supported for %s (404)", digestRef.String())
			return nil, nil
		}
		return nil, err
	}

	if index == nil || len(index.Manifests) == 0 {
		return nil, nil
	}

	manifests := index.Manifests
	if len(manifests) > maxReferrerManifests {
		log.Warnf("Image %s has %d referrer manifests, processing only first %d",
			digestRef.String(), len(manifests), maxReferrerManifests)
		manifests = manifests[:maxReferrerManifests]
	}

	var bundleDescs, legacyDescs []v1.Descriptor
	for _, desc := range manifests {
		switch {
		case strings.HasPrefix(desc.ArtifactType, bundleSigArtifactTypePrefix):
			bundleDescs = append(bundleDescs, desc)
		case desc.ArtifactType == cosignSigArtifactType:
			legacyDescs = append(legacyDescs, desc)
		}
	}

	// Try bundle-format referrers first, fall back to legacy format.
	payloads := extractPayloadsFromDescs(ctx, bundleDescs, repo, true, opts)
	if len(payloads) > 0 {
		log.Infof("Found %d payload(s) via bundle format for %s", len(payloads), digestRef.String())
		return payloads, nil
	}

	payloads = extractPayloadsFromDescs(ctx, legacyDescs, repo, false, opts)
	if len(payloads) > 0 {
		log.Infof("Found %d payload(s) via legacy format for %s", len(payloads), digestRef.String())
	}
	return payloads, nil
}

// extractPayloadsFromDescs iterates over pre-filtered referrer descriptors and extracts
// cosign signed payloads from each manifest.
func extractPayloadsFromDescs(ctx context.Context, descs []v1.Descriptor, repo name.Repository,
	isBundle bool, opts []ociremote.Option,
) []signaturePayload {
	var allPayloads []signaturePayload
	for _, desc := range descs {
		if ctx.Err() != nil {
			break
		}

		sigDigestRef := repo.Digest(desc.Digest.String())
		payloads, err := extractReferrerPayloads(isBundle, sigDigestRef, opts)
		if err != nil {
			log.Warnf("Failed to extract signatures from referrer %s: %v", desc.Digest, err)
			continue
		}
		allPayloads = append(allPayloads, payloads...)
	}
	return allPayloads
}

// extractReferrerPayloads fetches cosign signed payloads from a single referrer manifest.
func extractReferrerPayloads(isBundle bool, ref name.Reference, opts []ociremote.Option) ([]signaturePayload, error) {
	if isBundle {
		return fetchBundlePayloads(ref, opts)
	}
	sigs, err := ociremote.Signatures(ref, opts...)
	if err != nil {
		return nil, err
	}
	payloads, err := cosign.FetchSignatures(&referrerSignedEntity{sigs: sigs})
	return wrapPayloads(payloads), err
}

// fetchBundlePayloads fetches a sigstore bundle referrer and stores the raw bundle JSON
// for direct verification via sigstore-go at verify time.
func fetchBundlePayloads(bundleRef name.Reference, opts []ociremote.Option) ([]signaturePayload, error) {
	b, err := ociremote.Bundle(bundleRef, opts...)
	if err != nil {
		return nil, err
	}

	bundleJSON, err := b.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling sigstore bundle: %w", err)
	}

	return []signaturePayload{{
		signatureFormat: storage.CosignSignature_DEAD_SIMPLE_SIGNING_ENVELOPE,
		rawBundle:       bundleJSON,
	}}, nil
}
