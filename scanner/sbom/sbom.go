package sbom

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/claircore/sbom/spdx"
	"github.com/stackrox/rox/scanner/internal/version"
)

// Supported media types for SBOM encoding/decoding.
const (
	MediaTypeSPDXJSON = "application/spdx+json"
	MediaTypeSPDXText = "text/spdx+json"
)

// Options contains options for SBOM generation.
type Options struct {
	Name      string
	Namespace string
	Comment   string
}

// SBOMer handles both encoding (generation) and decoding (parsing) of SBOMs.
type SBOMer struct{}

// NewSBOMer creates a new SBOMer.
func NewSBOMer() *SBOMer {
	return &SBOMer{}
}

// GetSBOM encodes an IndexReport into an SBOM.
func (s *SBOMer) GetSBOM(ctx context.Context, ir *claircore.IndexReport, opts *Options) ([]byte, error) {
	if ir == nil {
		return nil, errors.New("index report is required")
	}

	if opts == nil {
		return nil, errors.New("opts is required")
	}

	encoder := spdx.NewDefaultEncoder(
		spdx.WithDocumentName(opts.Name),
		spdx.WithDocumentNamespace(opts.Namespace),
		spdx.WithDocumentComment(opts.Comment),
	)
	encoder.Creators = append(encoder.Creators, spdx.Creator{Creator: fmt.Sprintf("scanner-v4-matcher-%s", version.Version), CreatorType: "Tool"})
	encoder.Version = spdx.V2_3
	encoder.Format = spdx.JSONFormat

	b := &bytes.Buffer{}
	err := encoder.Encode(ctx, b, ir)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Decode decodes an SBOM into an IndexReport.
// The mediaType specifies the format of the SBOM (e.g., "application/spdx+json").
func (s *SBOMer) Decode(ctx context.Context, sbom []byte, mediaType string) (*claircore.IndexReport, error) {
	// Normalize media type by stripping parameters (e.g., charset).
	normalizedMediaType := NormalizeMediaType(mediaType)

	switch normalizedMediaType {
	case MediaTypeSPDXJSON, MediaTypeSPDXText:
		return s.decodeSPDX(ctx, sbom)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

// decodeSPDX decodes an SPDX JSON SBOM into an IndexReport.
func (s *SBOMer) decodeSPDX(ctx context.Context, sbom []byte) (*claircore.IndexReport, error) {
	decoder := spdx.NewDefaultDecoder()
	reader := bytes.NewReader(sbom)

	ir, err := decoder.Decode(ctx, reader)
	if err != nil {
		return nil, fmt.Errorf("decoding SPDX SBOM: %w", err)
	}

	return ir, nil
}

// NormalizeMediaType strips any parameters from the media type.
func NormalizeMediaType(mediaType string) string {
	return strings.TrimSpace(strings.Split(mediaType, ";")[0])
}

// SupportedMediaTypes returns the list of supported media types for SBOM decoding.
func SupportedMediaTypes() []string {
	return []string{MediaTypeSPDXJSON, MediaTypeSPDXText}
}

// IsSupportedMediaType checks if the given media type is supported.
func IsSupportedMediaType(mediaType string) bool {
	normalized := NormalizeMediaType(mediaType)
	for _, supported := range SupportedMediaTypes() {
		if normalized == supported {
			return true
		}
	}
	return false
}
