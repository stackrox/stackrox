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
	// mu mutex to avoid multiple goroutines writing at the same time
	mu sync.Mutex
}

// Init initializes the writer
func (tw *TraceWriter) Init() error {
	if path.Dir(tw.Destination) == "" {
		return nil
	}
	return os.MkdirAll(path.Dir(tw.Destination), os.ModePerm)
}

// Write a slice of bytes in the Destination file
func (tw *TraceWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
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
}

// Init initializes the reader
func (tw *TraceReader) Init() error {
	_, err := os.Stat(path.Dir(tw.Source))
	return err
}

// ReadFile reads the entire file and returns a slice of objects
func (tw *TraceReader) ReadFile() ([][]byte, error) {
	data, err := os.ReadFile(tw.Source)
	if err != nil {
		return nil, err
	}
	objs := bytes.Split(data, []byte{'\n'})
	return objs, nil
}
