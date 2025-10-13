package generators

import (
	"context"
)

// PathMapGenerator is a generator that produces a map from structured backup files/directories layout to its source.
//
//go:generate mockgen-wrapper
type PathMapGenerator interface {
	GeneratePathMap(ctx context.Context) (map[string]string, error)
}
