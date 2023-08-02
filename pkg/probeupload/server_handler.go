package probeupload

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

// ProbeSource is an interface that abstracts the functionality of loading a kernel probe.
type ProbeSource interface {
	// LoadProbe tries to load a probe, with `fileName` in the format `<module version>/<gzipped probe file>`.
	// If the size cannot be determined in advance, -1 should be returned as the second value.
	// A "not found" should be indicated via a `nil, 0, nil` return value.
	LoadProbe(ctx context.Context, fileName string) (io.ReadCloser, int64, error)

	IsAvailable(ctx context.Context) (bool, error)
}

type probeServerHandler struct {
	sources       []ProbeSource
	errorCallback func(error)
	centralReady  concurrency.Signal
}

// LogCallback returns an error callback that simply logs.
func LogCallback(logger logging.Logger) func(error) {
	return func(err error) {
		logger.Errorf("Error serving kernel probe: %v", err)
	}
}

// NewProbeServerHandler returns a http.Handler for serving kernel probes. The probeServerHandler assumes the path of kernel
// probes is rooted at `/`, i.e., wrap this via `http.StripPrefix` when serving on a sub-path.
// The errorCallback is invoked for errors that happen during writing the response body, and thus cannot be transmitted
// to the client via status/headers. It may be nil, in which case errors are simply ignored.
func NewProbeServerHandler(errorCallback func(error), sources ...ProbeSource) *probeServerHandler {
	return &probeServerHandler{
		errorCallback: errorCallback,
		sources:       sources,
		centralReady:  concurrency.NewSignal(),
	}
}

func (h *probeServerHandler) ComponentName() string {
	return "Kernel probe server handler"
}

func (h *probeServerHandler) GoOnline() {
	h.centralReady.Signal()
}

func (h *probeServerHandler) GoOffline() {
	h.centralReady.Reset()
}

func (h *probeServerHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		msg := fmt.Sprintf("invalid method %s, only %s requests are supported", req.Method, http.MethodGet)
		log.Error(msg)
		http.Error(w, msg, http.StatusMethodNotAllowed)
		return
	}

	if !strings.HasPrefix(req.URL.Path, "/") {
		msg := fmt.Sprintf("invalid path %q", req.URL.Path)
		log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	probePath := req.URL.Path[1:]
	log.Debugf("received request for probe at %s", probePath)

	if !IsValidFilePath(probePath) {
		log.Errorf("invalid probe path: %s", probePath)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var firstErr error
	var data io.ReadCloser
	var size int64
	for _, source := range h.sources {
		var err error
		data, size, err = source.LoadProbe(req.Context(), probePath)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if data != nil {
				_ = data.Close()
			}
			log.Errorf("error loading probe %s: %v", probePath, err)
		} else if data != nil {
			break
		}
	}

	if data == nil {
		if firstErr == nil {
			log.Infof("kernel probe %s not found", probePath)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, firstErr.Error(), http.StatusInternalServerError)
		return
	}
	defer utils.IgnoreError(data.Close)

	if !h.centralReady.IsDone() {
		log.Error("sensor is running in offline mode")
		http.Error(w, "sensor running in offline mode", http.StatusServiceUnavailable)
		return
	}

	hdr := w.Header()
	if size >= 0 { // size < 0 means unknown
		hdr.Set("Content-Length", strconv.FormatInt(size, 10))
	}
	hdr.Set("Content-Type", "application/octet-stream")
	hdr.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", path.Base(probePath)))
	w.WriteHeader(http.StatusOK)

	n, err := io.Copy(w, data)
	if err == nil && size >= 0 && n != size {
		err = errors.Errorf("read unexpected number of bytes: got %d, expected %d", n, size)
	}
	if err != nil && h.errorCallback != nil {
		h.errorCallback(err)
	}
}
