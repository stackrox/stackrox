package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
)

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

type traceReader struct {
	source  string
	mode    string
	lineNo  int64
	enabled bool
	mu      sync.Mutex
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

func (tw *traceReader) ReadNextMsg() (resources.InformerK8sMsg, error) {
	if !tw.enabled {
		return resources.InformerK8sMsg{}, nil
	}

	file, err := os.OpenFile(tw.source, os.O_RDONLY, 0644)
	if err != nil {
		return resources.InformerK8sMsg{}, errors.Wrapf(err, "Error opening file: %s\n", tw.source)
	}
	defer func() {
		_ = file.Close()
	}()
	tw.mu.Lock()
	defer tw.mu.Unlock()
	scanner := bufio.NewScanner(file)
	obj := resources.InformerK8sMsg{}
	var lno int64
	for scanner.Scan() {
		lno++
		if lno < tw.lineNo {
			continue
		}
		if err := json.Unmarshal(scanner.Bytes(), &obj); err != nil {
			tw.lineNo = lno
			return resources.InformerK8sMsg{}, errors.Wrap(err, "error when unmarshalling")
		}
		return obj, nil
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "shouldn't see an error scanning a string")
	}
	return resources.InformerK8sMsg{}, nil
}
