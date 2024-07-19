package io

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/stackrox/rox/pkg/env"
)

// IO holds information about io streams used within commands of roxctl.
//
//go:generate mockgen-wrapper
type IO interface {
	In() io.Reader
	Out() io.Writer
	ErrOut() io.Writer
}

type ioImpl struct {
	// In = os.Stdin
	in io.Reader
	// Out = os.Stdout
	out io.Writer
	// ErrOut = os.Stderr
	errOut io.Writer
}

func (i *ioImpl) In() io.Reader {
	return i.in
}

func (i *ioImpl) Out() io.Writer {
	return i.out
}

func (i *ioImpl) ErrOut() io.Writer {
	return i.errOut
}

// getOutputFile opens a file in write (append) mode for output or error writing.
// A nil output means that the specified path is either empty, or
// that something prevented the file from being opened in write mode.
func getOutputFile(filePath string) *os.File {
	if len(filePath) == 0 {
		return nil
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil && !os.IsNotExist(err) {
		return nil
	}
	if os.IsNotExist(err) {
		// If the output file does not exist, try to create it along with the path to it.
		dirPath := filepath.Dir(filePath)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			// Could not create directory to store the output file
			// Revert to default output
			return nil
		}
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
		if err != nil {
			return nil
		}
		return file
	}
	if fileInfo.Mode()&0200 != 0 {
		file, err := os.OpenFile(filePath, os.O_RDWR|os.O_APPEND, 0600)
		if err != nil {
			return nil
		}
		return file
	}
	return nil
}

// DefaultIO uses the default os specific streams for input / output
func DefaultIO() IO {
	readerAndWriters := &ioImpl{
		in:     os.Stdin,
		out:    os.Stdout,
		errOut: os.Stderr,
	}
	if newOutput := getOutputFile(env.OutputFile.Setting()); newOutput != nil {
		readerAndWriters.out = newOutput
	}
	if newErrorOutput := getOutputFile(env.ErrorFile.Setting()); newErrorOutput != nil {
		readerAndWriters.errOut = newErrorOutput
	}
	return readerAndWriters
}

// DiscardIO discards IO.Out and IO.ErrOut
// This is especially useful during testing when output is non-relevant and shall be suppressed
func DiscardIO() IO {
	return &ioImpl{
		in:     os.Stdin,
		out:    io.Discard,
		errOut: io.Discard,
	}
}

// TestIO creates an IO and returns *bytes.Buffer for IO.In, IO.Out and IO.ErrOut respectively
// This is especially useful during testing when input / output is relevant and needs to be evaluated
func TestIO() (IO, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	return &ioImpl{
		in:     in,
		out:    out,
		errOut: errOut,
	}, in, out, errOut
}
