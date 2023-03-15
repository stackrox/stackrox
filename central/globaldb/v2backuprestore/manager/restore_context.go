package manager

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
)

type restoreProcessContext struct {
	context.Context

	outputDir string

	numAsyncChecks int
	asyncErrorsC   chan error

	postgresBundle bool
}

func newRestoreProcessContext(ctx context.Context, outputDir string, postgresBundle bool) *restoreProcessContext {
	return &restoreProcessContext{
		Context:        ctx,
		outputDir:      strings.TrimRight(outputDir, "/") + "/",
		asyncErrorsC:   make(chan error),
		postgresBundle: postgresBundle,
	}
}

func (c *restoreProcessContext) OutputDir() string {
	return c.outputDir
}

func (c *restoreProcessContext) ResolvePath(relativePath string) (string, error) {
	path := filepath.Join(c.outputDir, relativePath)
	if !strings.HasPrefix(path, c.outputDir) || strings.Contains(path, "..") {
		return "", errors.Errorf("path %q is not a sub path of %s", path, c.outputDir)
	}
	return path, nil
}

func (c *restoreProcessContext) OpenFile(relativePath string, flags int, perm os.FileMode) (*os.File, error) {
	path, err := c.ResolvePath(relativePath)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(path, flags, perm)
}

func (c *restoreProcessContext) Mkdir(relativePath string, perm os.FileMode) (string, error) {
	path, err := c.ResolvePath(relativePath)
	if err != nil {
		return "", err
	}
	if err := os.Mkdir(path, perm); err != nil {
		return "", err
	}
	return path, nil
}

func (c *restoreProcessContext) IsPostgresBundle() bool {
	return c.postgresBundle
}

func (c *restoreProcessContext) dispatchAsyncCheck(checkFn func(ctx common.RestoreProcessContext) error, fileName string) {
	c.numAsyncChecks++
	go c.runAsyncCheck(checkFn, fileName)
}

func (c *restoreProcessContext) runAsyncCheck(checkFn func(ctx common.RestoreProcessContext) error, fileName string) {
	err := errors.Wrapf(checkFn(c), "error running asynchronous check for file %s", fileName)
	select {
	case c.asyncErrorsC <- err:
	case <-c.Done():
	}
}

func (c *restoreProcessContext) forFile(fileName string) *restoreFileContext {
	return &restoreFileContext{
		restoreProcessContext: c,
		fileName:              fileName,
	}
}

func (c *restoreProcessContext) waitForAsyncChecks() error {
	for i := 0; i < c.numAsyncChecks; i++ {
		select {
		case asyncErr := <-c.asyncErrorsC:
			if asyncErr != nil {
				return asyncErr
			}
		case <-c.Done():
			return errors.Wrap(c.Err(), "context error waiting for asynchronous checks")
		}
	}
	return nil
}

type restoreFileContext struct {
	*restoreProcessContext

	fileName string
}

func (c *restoreFileContext) FileName() string {
	return c.fileName
}

func (c *restoreFileContext) CheckAsync(checkFn func(ctx common.RestoreProcessContext) error) {
	c.restoreProcessContext.dispatchAsyncCheck(checkFn, c.fileName)
}
