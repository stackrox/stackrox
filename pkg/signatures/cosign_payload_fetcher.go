package signatures

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	"github.com/sigstore/cosign/v3/pkg/oci"
	ociremote "github.com/sigstore/cosign/v3/pkg/oci/remote"
	"github.com/stackrox/rox/generated/storage"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
)

const cosignSigArtifactType = "application/vnd.dev.cosign.artifact.sig.v1+json"

// maxReferrerManifests limits the number of referrer signature manifests processed per image
// to bound latency and prevent pathological cases from consuming the caller's timeout budget.
const maxReferrerManifests = 50

var (
	_ oci.SignedEntity = (*tagSignedEntity)(nil)
	_ oci.SignedEntity = (*referrerSignedEntity)(nil)
)

// tagSignedEntity adapts an image reference for tag-based signature discovery.
// Signatures() constructs the cosign tag reference (<algo>-<hex>.sig) and fetches
// the signature manifest from the registry. Digest() returns the image digest so
// cosign can build the tag. Only these two methods are called by cosign.FetchSignatures.
type tagSignedEntity struct {
	oci.SignedEntity
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

// referrerSignedEntity adapts pre-fetched referrer signatures for cosign.FetchSignatures.
// Signatures() returns the already-fetched data. Only this method is called by
// cosign.FetchSignatures — other SignedEntity methods are intentionally unimplemented.
type referrerSignedEntity struct {
	oci.SignedEntity
	sigs oci.Signatures
}

func (r *referrerSignedEntity) Signatures() (oci.Signatures, error) {
	return r.sigs, nil
}

// fetchTagPayloads discovers signatures via the legacy cosign tag-based method.
// It uses a local signed entity to skip fetching the image manifest and only fetch the signature manifest.
// When no signature is associated with the image, cosign returns an error which is converted to nil
// because "no signatures" is a valid state. Since cosign does not expose typed errors, we use string
// comparison (see isMissingSignatureError).
// Cosign ref: https://github.com/sigstore/cosign/blob/main/pkg/cosign/fetch.go
func fetchTagPayloads(image *storage.Image, imgRef name.Reference, opts []ociremote.Option) ([]cosign.SignedPayload, error) {
	se := newTagSignedEntity(image, imgRef, opts...)
	payloads, err := cosign.FetchSignatures(se)
	if err != nil && (isMissingSignatureError(err) || isUnknownMimeTypeError(err)) {
		return nil, nil
	}
	return payloads, err
}

// fetchReferrerPayloads queries the OCI 1.1 Referrers API for cosign signature artifacts
// attached to the given image digest and returns them as cosign signed payloads.
// ctx is checked between manifest iterations to bail out early on cancellation.
func fetchReferrerPayloads(ctx context.Context, digestRef name.Digest, repo name.Repository,
	opts []ociremote.Option,
) ([]cosign.SignedPayload, error) {
	log.Infof("Fetching OCI referrer signatures for %s with artifact type %s",
		digestRef.String(), cosignSigArtifactType)

	index, err := ociremote.Referrers(digestRef, cosignSigArtifactType, opts...)
	if err != nil {
		// A 404 means the registry does not support the OCI 1.1 referrers endpoint.
		// This is expected and not an error — the caller falls back to tag-based discovery.
		// The heroku ErrorTransport converts 404 responses into Go errors before
		// go-containerregistry's built-in fallback can handle them, so we catch it here.
		if checkIfErrorContainsCode(err, http.StatusNotFound) {
			log.Infof("OCI referrers API not supported for %s (404), falling back to tag-based discovery",
				digestRef.String())
			return nil, nil
		}
		log.Infof("OCI referrers API call failed for %s: %v", digestRef.String(), err)
		return nil, err
	}

	if index == nil || len(index.Manifests) == 0 {
		log.Infof("No OCI referrer manifests found for %s", digestRef.String())
		return nil, nil
	}

	manifests := index.Manifests
	if len(manifests) > maxReferrerManifests {
		log.Warnf("Image %s has %d referrer signature manifests, processing only the first %d",
			digestRef.String(), len(manifests), maxReferrerManifests)
		manifests = manifests[:maxReferrerManifests]
	}
	log.Infof("Found %d OCI referrer manifests for %s", len(manifests), digestRef.String())

	var allPayloads []cosign.SignedPayload
	for _, desc := range manifests {
		if ctx.Err() != nil {
			log.Infof("Context cancelled, stopping referrer processing for %s", digestRef.String())
			break
		}
		log.Infof("Processing referrer manifest %s for %s", desc.Digest, digestRef.String())
		sigDigestRef := repo.Digest(desc.Digest.String())
		sigs, err := ociremote.Signatures(sigDigestRef, opts...)
		if err != nil {
			log.Warnf("Skipping referrer signature manifest %s: %v", desc.Digest, err)
			continue
		}

		se := &referrerSignedEntity{sigs: sigs}
		payloads, err := cosign.FetchSignatures(se)
		if err != nil {
			log.Warnf("Failed to extract signatures from referrer %s: %v", desc.Digest, err)
			continue
		}
		log.Infof("Extracted %d payload(s) from referrer manifest %s", len(payloads), desc.Digest)
		allPayloads = append(allPayloads, payloads...)
	}

	log.Infof("Fetched %d total cosign payload(s) via OCI referrers API for %s",
		len(allPayloads), digestRef.String())
	return allPayloads, nil
}
