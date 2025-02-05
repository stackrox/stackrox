package sbom

import (
	"bytes"
	"context"
	"fmt"

	"github.com/quay/claircore"
	"github.com/quay/claircore/sbom/spdx"
	"github.com/stackrox/rox/scanner/internal/version"
)

type SBOMer struct {
}

func NewSBOMer() *SBOMer {
	return &SBOMer{}
}

func (s *SBOMer) GetSBOM(ctx context.Context, ir *claircore.IndexReport, name, id string) ([]byte, error) {
	encoder := spdx.NewDefaultEncoder(
		spdx.WithDocumentName(id),
		spdx.WithDocumentNamespace("TODO NAMESPACE"),
		spdx.WithDocumentComment(fmt.Sprintf("Tech Preview - generated for '%s'", name)),
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
