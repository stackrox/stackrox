package environment

import (
	"bytes"
	"io"
	"os"
)

// IO holds information about io streams used within commands of roxctl
type IO struct {
	// In = os.Stdin
	In io.Reader
	// Out = os.Stdout
	Out io.Writer
	// ErrOut = os.Stderr
	ErrOut io.Writer
}

// DefaultIO uses the default os specific streams for input / output
func DefaultIO() IO {
	return IO{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

// DiscardIO discards IO.Out and IO.ErrOut
// This is especially useful during testing when output is non-relevant and shall be suppressed
func DiscardIO() IO {
	return IO{
		In:     os.Stdin,
		Out:    io.Discard,
		ErrOut: io.Discard,
	}
}

// TestIO creates an IO and returns *bytes.Buffer for IO.In, IO.Out and IO.ErrOut respectively
// This is especially useful during testing when input / output is relevant and needs to be evaluated
func TestIO() (IO, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	return IO{
		In:     in,
		Out:    out,
		ErrOut: errOut,
	}, in, out, errOut
}
