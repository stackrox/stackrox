package k8s

import (
	"bytes"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

var _ io.Writer = (*TraceWriter)(nil)

// TraceWriter writes sensor handled k8s events into a file
type TraceWriter struct {
	// Destination file where we will store the events
	Destination string
	// Enabled defines whether this writer should do anything at all
	Enabled bool
}

// Init initializes the writer
func (tw *TraceWriter) Init() error {
	if !tw.Enabled || path.Dir(tw.Destination) == "" {
		return nil
	}
	return os.MkdirAll(path.Dir(tw.Destination), os.ModePerm)
}

// Write a slice of bytes in the Destination file
func (tw *TraceWriter) Write(b []byte) (int, error) {
	if !tw.Enabled {
		return 0, nil
	}
	fObjs, err := os.OpenFile(tw.Destination, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, errors.Wrapf(err, "Error opening file: %s\n", tw.Destination)
	}
	defer func() {
		_ = fObjs.Close()
	}()
	return fObjs.Write(append(b, []byte{10}...))
}

// TraceReader reads a file containing k8s events
type TraceReader struct {
	// Source file from which the lines are read
	Source string
	// lineNo is the pointer marking which line from the file has been read recently
	lineNo int64
	// Enabled defines whether this reader should do anything at all (can be removed maybe)
	Enabled bool
	// mu is a mutex that might be useful if many goroutines would read from the same file
	mu sync.Mutex
}

// Init initializes the reader
func (tw *TraceReader) Init() error {
	if !tw.Enabled || path.Dir(tw.Source) == "" {
		return nil
	}
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.lineNo = 1
	_, err := os.Stat(path.Dir(tw.Source))
	return err
}

// ReadFile reads the entire file and returns a slice of objects
func (tw *TraceReader) ReadFile() ([][]byte, error) {
	if !tw.Enabled {
		return nil, nil
	}
	data, err := os.ReadFile(tw.Source)
	if err != nil {
		return nil, err
	}
	objs := bytes.Split(data, []byte{'\n'})
	return objs, nil
}
