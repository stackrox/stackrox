package sbomer

import (
	"context"

	"github.com/quay/claircore"
)

type SBOMer interface {
	GetSBOM(ctx context.Context, ir *claircore.IndexReport) ([]byte, error)
}

type sbomerImpl struct {
}

var _ SBOMer = (*sbomerImpl)(nil)

func NewSBOMer(ctx context.Context) SBOMer {
	// initializing libsbom here (if init needed)

	return &sbomerImpl{}
}

// GetSBOM implements SBOMer.
func (s *sbomerImpl) GetSBOM(ctx context.Context, ir *claircore.IndexReport) ([]byte, error) {
	panic("unimplemented")
}
