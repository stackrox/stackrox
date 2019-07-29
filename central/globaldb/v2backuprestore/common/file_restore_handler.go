package common

import (
	"io"
)

// RestoreFileHandlerFunc is a function that is invoked for restoring a file from a database export.
type RestoreFileHandlerFunc func(ctx RestoreFileContext, fileReader io.Reader, size int64) error

// FileHandlerDesc describes how a file is handled for database export.
type FileHandlerDesc struct {
	fileName           string
	optional           bool
	restoreHandlerFunc RestoreFileHandlerFunc
}

// NewFileHandler returns a new file handler description.
func NewFileHandler(fileName string, optional bool, restoreHandlerFunc RestoreFileHandlerFunc) *FileHandlerDesc {
	return &FileHandlerDesc{
		fileName:           fileName,
		optional:           optional,
		restoreHandlerFunc: restoreHandlerFunc,
	}
}

// FileName returns the file name of the file handled by this handler.
func (d *FileHandlerDesc) FileName() string {
	return d.fileName
}

// Optional returns whether the described file is optional.
func (d *FileHandlerDesc) Optional() bool {
	return d.optional
}

// RestoreHandlerFunc returns the handler function for this file.
func (d *FileHandlerDesc) RestoreHandlerFunc() RestoreFileHandlerFunc {
	return d.restoreHandlerFunc
}
