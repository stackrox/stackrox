package k8s

import (
	"bytes"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

var _ io.Writer = (*TraceWriter)(nil)

// TraceWriter writes sensor handled k8s events into a file
type TraceWriter struct {
	// mu mutex to avoid multiple goroutines writing at the same time
	mu sync.Mutex
	f  *os.File
}

// NewTraceWriter initializes the writer with destination file where we will store the events
func NewTraceWriter(dest string) (*TraceWriter, error) {
	dir := path.Dir(dest)
	if dir == "" {
		return nil, errors.New("trace destination directory must be set")
	}
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, errors.Wrap(err, "error creating trace destination directory")
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening trace destination file: %s", dest)
	}

	return &TraceWriter{
		f: f,
	}, nil
}

// Close closes the file
func (tw *TraceWriter) Close() error {
	return errors.Wrap(tw.f.Close(), "error closing trace writer")
}

var delimiter = []byte{'\n'}

// Write a slice of bytes in the Destination file
func (tw *TraceWriter) Write(b []byte) (int, error) {
	return concurrency.WithLock2(&tw.mu, func() (int, error) {
		n, err := tw.f.Write(b)
		if err != nil {
			return n, err
		}
		m, err := tw.f.Write(delimiter)
		if err != nil {
			return n + m, err
		}
		err = tw.f.Sync()
		return n + m, err
	})
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
	objs := bytes.Split(data, delimiter)
	return objs, nil
}
