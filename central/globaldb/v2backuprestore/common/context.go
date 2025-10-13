package common

import (
	"context"
	"os"
)

// RestoreProcessContext is the context active during a database export restore process.
type RestoreProcessContext interface {
	context.Context

	OutputDir() string
	ResolvePath(relativePath string) (string, error)
	OpenFile(relativePath string, flags int, perm os.FileMode) (*os.File, error)
	Mkdir(relativePath string, perm os.FileMode) (string, error)
	IsPostgresBundle() bool
}

// RestoreFileContext is the context active during the restoration of a single file from a database export.
type RestoreFileContext interface {
	RestoreProcessContext

	FileName() string

	// CheckAsync performs the given check asynchronously.
	CheckAsync(checkFn func(ctx RestoreProcessContext) error)
}
