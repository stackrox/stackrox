package main

import (
	"bufio"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

var _ io.Writer = (*traceWriter)(nil)

type traceWriter struct {
	destination string
	enabled     bool
}

func (tw *traceWriter) Init() error {
	if !tw.enabled || path.Dir(tw.destination) == "" {
		return nil
	}
	return os.MkdirAll(path.Dir(tw.destination), os.ModePerm)
}

func (tw *traceWriter) Write(b []byte) (int, error) {
	if !tw.enabled {
		return 0, nil
	}
	fObjs, err := os.OpenFile(tw.destination, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, errors.Wrapf(err, "Error opening file: %s\n", tw.destination)
	}
	defer func() {
		_ = fObjs.Close()
	}()
	return fObjs.Write(append(b, []byte{10}...))
}

var _ io.Reader = (*traceReader)(nil)

type traceReader struct {
	// source file from which the lines are read
	source string
	// mode defines whether to wait between replaying the consecutive k8s events
	mode string
	// lineNo is the pointer marking which line from the file has been read recently
	lineNo int64
	// enabled defines whether this reader should do anything at all (can be removed maybe)
	enabled bool
	// mu is a mutex that might be useful if many goroutines would read from the same file
	mu sync.Mutex
}

func (tw *traceReader) Init() error {
	if !tw.enabled || path.Dir(tw.source) == "" {
		return nil
	}
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.lineNo = 0
	return os.MkdirAll(path.Dir(tw.source), os.ModePerm)
}

// Read reads one line from the trace file. This line corresponds to a single K8s event
func (tw *traceReader) Read(p []byte) (n int, err error) {
	if !tw.enabled {
		return 0, nil
	}
	file, err := os.OpenFile(tw.source, os.O_RDONLY, 0644)
	if err != nil {
		return 0, errors.Wrapf(err, "Error opening file: %s\n", tw.source)
	}
	defer func() {
		_ = file.Close()
	}()
	tw.mu.Lock()
	defer tw.mu.Unlock()
	scanner := bufio.NewScanner(file)
	var lno int64
	for scanner.Scan() {
		lno++
		if lno < tw.lineNo {
			continue
		}
		b := scanner.Bytes()
		n = copy(p, b)
		return n, scanner.Err()
	}
	return 0, scanner.Err()
}
