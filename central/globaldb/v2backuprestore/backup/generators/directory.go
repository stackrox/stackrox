package generators

import (
	"context"
)

// DirectoryGenerator is a generator that produces a backup in the form of a directory of files.
//
//go:generate mockgen-wrapper
type DirectoryGenerator interface {
	WriteDirectory(ctx context.Context) (string, error)
}
