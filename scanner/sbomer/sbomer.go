package sbomer

import (
	"context"
	"io"

	"github.com/quay/claircore"
	"github.com/quay/claircore/sbom/spdx"
)

//go:generate mockgen-wrapper
type SBOMer interface {
	// GetSBOM generates an SBOM from an index report.
	GetSBOM(ctx context.Context, imageDigest claircore.Digest, ir *claircore.IndexReport) ([]byte, error)
}

type sbomerImpl struct {
}

var _ SBOMer = (*sbomerImpl)(nil)

func NewSBOMer(ctx context.Context) SBOMer {
	return &sbomerImpl{}
}

func (s *sbomerImpl) GetSBOM(ctx context.Context, imageDigest claircore.Digest, ir *claircore.IndexReport) ([]byte, error) {
	encoder := spdx.Encoder{
		Version: spdx.V2_3,
		Format:  spdx.JSON,
		Creators: []spdx.Creator{
			{Creator: "David Caravello", CreatorType: "Person"},
			{Creator: "Brad Lugo", CreatorType: "Person"},
			{Creator: "David Vail", CreatorType: "Person"},
			{Creator: "Surabhi LNU", CreatorType: "Person"},
		},
		DocumentName:      imageDigest.String(),
		DocumentNamespace: "DocumentNamespace?",
		DocumentComment:   "Bug Bash Demo",
	}

	reader, err := encoder.Encode(ctx, ir)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(reader)
}
