package common

import (
	"bytes"
	"io"
	"os"
)

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

// DefaultIO uses the default os specific streams for input / output
func DefaultIO() IO {
	return &ioImpl{
		in:     os.Stdin,
		out:    os.Stdout,
		errOut: os.Stderr,
	}
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
