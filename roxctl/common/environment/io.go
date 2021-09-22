package environment

import (
	"io"
	"os"
)

//IO holds information about io streams used within commands of roxctl
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
