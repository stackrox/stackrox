package sbom

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/package-url/packageurl-go"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/claircore/purl"
	"github.com/quay/claircore/python"
	"github.com/quay/claircore/rhel"
	"github.com/quay/claircore/sbom"
	"github.com/quay/claircore/sbom/spdx"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
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

// RepositoryToCPEProvider provides repository-to-CPE mappings.
// Both Indexer and RemoteIndexer implement this interface.
type RepositoryToCPEProvider interface {
	GetRepositoryToCPEMapping(ctx context.Context) (*repositorytocpe.MappingFile, error)
}

// SBOMer handles both encoding (generation) and decoding (parsing) of SBOMs.
type SBOMer struct {
	decoder sbom.Decoder
}

// NewSBOMer creates a new SBOMer with the given repository-to-CPE provider.
// The provider is used to fetch CPE information for RPM packages during SBOM decoding.
func NewSBOMer(repo2cpeProvider RepositoryToCPEProvider) *SBOMer {
	reg := purl.NewRegistry()
	reg.RegisterPurlType(python.PURLType, purl.NoneNamespace, python.ParsePURL)
	if repo2cpeProvider != nil {
		reg.RegisterPurlType(rhel.PURLType, rhel.PURLNamespace, rhel.ParseRPMPURL, repo2CPETransformer(repo2cpeProvider))
	} else if features.SBOMScanning.Enabled() {
		zlog.Warn(context.Background()).Msg("no repositoryToCPE provider configured")
	}

	// Only support SPDX decoding.
	decoder := spdx.NewDefaultDecoder(spdx.WithDecoderPURLConverter(reg))

	return &SBOMer{decoder}
}

// repo2CPETransformer creates a TransformerFunc that adds repository CPEs to RPM PURLs.
// It looks up the repository_id qualifier and adds the repository_cpes qualifier
// with the corresponding CPEs from the provider.
func repo2CPETransformer(provider RepositoryToCPEProvider) purl.TransformerFunc {
	return func(ctx context.Context, p *packageurl.PackageURL) error {
		if provider == nil {
			return nil
		}

		qualifiersMap := p.Qualifiers.Map()
		repoID, ok := qualifiersMap[rhel.PURLRepositoryID]
		if !ok {
			return nil
		}

		repo2cpe, err := provider.GetRepositoryToCPEMapping(ctx)
		if err != nil || repo2cpe == nil {
			// Best effort - continue without CPE enrichment.
			return nil
		}

		cpes, ok := repo2cpe.GetCPEs(repoID)
		if !ok || len(cpes) == 0 {
			zlog.Debug(ctx).Msgf("could not find repoid \"%s\" in cpe mapping", repoID)
			// Repository not in mapping.
			return nil
		}

		// Add the repository_cpes qualifier as a comma-separated list.
		// Don't overwrite if it's already set.
		if _, exists := qualifiersMap[rhel.PURLRepositoryCPEs]; exists {
			zlog.Debug(ctx).Msgf("found extra CPEs (not recorded): %s", strings.Join(cpes, ", "))
			return nil
		}

		qualifiersMap[rhel.PURLRepositoryCPEs] = strings.Join(cpes, ",")
		p.Qualifiers = packageurl.QualifiersFromMap(qualifiersMap)

		return nil
	}
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
func (s *SBOMer) Decode(ctx context.Context, sbomData []byte, mediaType string) (*claircore.IndexReport, error) {
	switch mediaType {
	case MediaTypeSPDXJSON, MediaTypeSPDXText:
		return s.decodeSPDX(ctx, sbomData)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

// decodeSPDX decodes an SPDX JSON SBOM into an IndexReport.
func (s *SBOMer) decodeSPDX(ctx context.Context, sbomData []byte) (*claircore.IndexReport, error) {
	reader := bytes.NewReader(sbomData)

	ir, err := s.decoder.Decode(ctx, reader)
	if err != nil {
		return nil, fmt.Errorf("decoding SPDX SBOM: %w", err)
	}

	return ir, nil
}
