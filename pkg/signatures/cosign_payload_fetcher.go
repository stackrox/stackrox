package signatures

import (
	"context"
	"errors"
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
	bundleSigArtifactTypePrefix = "application/vnd.dev.sigstore.bundle"

	// maxReferrerManifests limits the number of referrer signature manifests processed per image
	// to bound latency and prevent pathological cases from consuming the caller's timeout budget.
	maxReferrerManifests = 50
)

// signaturePayload is the internal representation of a fetched cosign signature before
// it is persisted into the CosignSignature proto. Two formats are supported:
//
//   - SimpleSigning: the legacy cosign format. The signature, payload, certificate, and
//     rekor bundle are stored as individual fields in cosign.SignedPayload.
//   - Sigstore bundle: the OCI 1.1 bundle format. The raw bundle JSON is stored as-is
//     in sigstoreBundle and verified directly via sigstore-go at verification time.
//
// The format is determined by sigstoreBundle: non-empty means sigstore bundle,
// empty means SimpleSigning.
type signaturePayload struct {
	cosign.SignedPayload
	sigstoreBundle []byte
}

var _ oci.SignedEntity = (*tagSignedEntity)(nil)

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
	return nil, errors.New("attachments not supported on tag-based signature entity")
}

// fetchSignaturesByTag discovers SimpleSigning signatures via the cosign tag-based method.
// Discovery: looks up the tag <algo>-<hex>.sig in the same repository as the image.
// Format: always SimpleSigning (signature and payload stored in OCI layer annotations).
func fetchSignaturesByTag(image *storage.Image, imgRef name.Reference, opts []ociremote.Option) ([]signaturePayload, error) {
	se := newTagSignedEntity(image, imgRef, opts...)
	payloads, err := cosign.FetchSignatures(se)
	if err != nil && (isMissingSignatureError(err) || isUnknownMimeTypeError(err)) {
		return nil, nil
	}
	wrapped := make([]signaturePayload, len(payloads))
	for i, p := range payloads {
		wrapped[i] = signaturePayload{SignedPayload: p}
	}
	return wrapped, err
}

// fetchSignaturesByReferrer discovers sigstore bundle signatures via the OCI 1.1 Referrers API.
// Discovery: queries the referrers index for the image digest and filters for sigstore
// bundle artifact types. Only the sigstore bundle format is supported for referrer-based
// discovery — cosign v3 exclusively produces bundles for this path.
func fetchSignaturesByReferrer(ctx context.Context, digestRef name.Digest, repo name.Repository,
	opts []ociremote.Option,
) ([]signaturePayload, error) {
	index, err := ociremote.Referrers(digestRef, "", opts...)
	if err != nil {
		if checkIfErrorContainsCode(err, http.StatusNotFound) {
			log.Warnf("OCI referrers API not supported for %s (404)", digestRef.String())
			return nil, nil
		}
		return nil, err
	}
	if index == nil {
		return nil, nil
	}

	manifests := index.Manifests
	if len(manifests) > maxReferrerManifests {
		log.Warnf("Image %s has %d referrer manifests, processing only first %d",
			digestRef.String(), len(manifests), maxReferrerManifests)
		manifests = manifests[:maxReferrerManifests]
	}

	var payloads []signaturePayload
	for _, desc := range manifests {
		if ctx.Err() != nil {
			break
		}
		if !strings.HasPrefix(desc.ArtifactType, bundleSigArtifactTypePrefix) {
			continue
		}
		p, err := fetchSigstoreBundle(repo.Digest(desc.Digest.String()), opts)
		if err != nil {
			log.Warnf("Failed to fetch sigstore bundle from referrer %s: %v", desc.Digest, err)
			continue
		}
		payloads = append(payloads, p...)
	}
	return payloads, nil
}

// fetchSigstoreBundle fetches a sigstore bundle referrer and stores the raw JSON.
// Format: sigstore bundle (DSSE envelope + verification material + tlog entries in one blob).
// Storage: the raw bundle JSON is stored in SigstoreBundle; no decomposition into individual
// fields. Verification is deferred to verifySigstoreBundle which uses sigstore-go directly.
func fetchSigstoreBundle(bundleRef name.Reference, opts []ociremote.Option) ([]signaturePayload, error) {
	b, err := ociremote.Bundle(bundleRef, opts...)
	if err != nil {
		return nil, err
	}

	bundleJSON, err := b.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling sigstore bundle: %w", err)
	}

	return []signaturePayload{{
		sigstoreBundle: bundleJSON,
	}}, nil
}
