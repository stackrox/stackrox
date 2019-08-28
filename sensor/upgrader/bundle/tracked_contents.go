package bundle

import (
	"io"

	"github.com/stackrox/rox/pkg/set"
)

type trackedContents struct {
	Contents
	openedFiles set.StringSet
}

// trackContents allows tracking which files have been opened.
func trackContents(c Contents) *trackedContents {
	return &trackedContents{
		Contents:    c,
		openedFiles: set.NewStringSet(),
	}
}

func (c *trackedContents) File(fileName string) OpenFunc {
	openFn := c.Contents.File(fileName)
	if openFn == nil {
		return nil
	}

	return func() (io.ReadCloser, error) {
		c.openedFiles.Add(fileName)
		return openFn()
	}
}

func (c *trackedContents) OpenedFiles() []string {
	return c.openedFiles.AsSlice()
}
