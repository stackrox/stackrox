package matcher

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

//go:generate mockgen-wrapper
type Matcher interface {
	MatchWithBaseImages(ctx context.Context, layers []string) ([]*storage.BaseImage, error)
}
