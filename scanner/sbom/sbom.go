package sbom

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/claircore/sbom/spdx"
	"github.com/stackrox/rox/scanner/internal/version"
)

type SBOMer struct {
}

type Options struct {
	Name      string
	Namespace string
	Comment   string
}

func NewSBOMer() *SBOMer {
	return &SBOMer{}
}

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
