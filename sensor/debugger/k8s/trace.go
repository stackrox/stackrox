package k8s

import (
	"bufio"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

var _ io.Writer = (*TraceWriter)(nil)

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

var _ io.Reader = (*TraceReader)(nil)

type TraceReader struct {
	// Source file from which the lines are read
	Source string
	// mode defines whether to wait between replaying the consecutive k8s events
	mode string
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
	return os.MkdirAll(path.Dir(tw.Source), os.ModePerm)
}

// ReadFile reads the file line by line and executes the handle function
func (tw *TraceReader) ReadFile(mode CreateMode, done chan int, handle func([]byte, CreateMode)) error {
	if !tw.Enabled {
		return nil
	}
	for {
		buf := make([]byte, 8*4096)
		n, err := tw.Read(buf)
		if err != nil {
			done <- 0
			return err
		}
		handle(buf[:n], mode)
	}
}

// Read reads one line from the trace file. This line corresponds to a single K8s event
func (tw *TraceReader) Read(p []byte) (n int, err error) {
	if !tw.Enabled {
		return 0, nil
	}
	file, err := os.OpenFile(tw.Source, os.O_RDONLY, 0644)
	if err != nil {
		return 0, errors.Wrapf(err, "Error opening file: %s\n", tw.Source)
	}
	defer func() {
		_ = file.Close()
	}()
	tw.mu.Lock()
	defer tw.mu.Unlock()
	scanner := bufio.NewScanner(file)
	scBuf := make([]byte, 0, 64*1024)
	scanner.Buffer(scBuf, 1024*1024)
	var lno int64
	for scanner.Scan() {
		lno++
		if lno < tw.lineNo {
			continue
		}
		tw.lineNo++
		b := scanner.Bytes()
		n = copy(p, b)
		return n, scanner.Err()
	}
	return 0, io.EOF
}

// ReadFileBlocking reads the file line by line blocking the mutex until is done
func (tw *TraceReader) ReadFileBlocking(mode CreateMode, done chan int, handle func([]byte, CreateMode)) error {
	if !tw.Enabled {
		return nil
	}
	file, err := os.OpenFile(tw.Source, os.O_RDONLY, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error opening file: %s\n", tw.Source)
	}
	defer func() {
		_ = file.Close()
	}()
	tw.mu.Lock()
	defer tw.mu.Unlock()
	buf := bufio.NewReader(file)
	for {
		line, err := buf.ReadBytes('\n')
		if err != nil {
			done <- 0
			return err
		}
		handle(line, mode)
	}
}
