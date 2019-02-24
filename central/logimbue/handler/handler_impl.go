package handler

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/stackrox/rox/central/logimbue/store"
)

type handlerImpl struct {
	storage            store.Store
	compressorProvider func() (Compressor, error)
}

// ServeHTTP adds or retrieves logs from the backend
func (l handlerImpl) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	// If we panic unpacking the contents, we want to return an HTTP error for a bad request.
	defer recoverFromBuffPanic(resp)

	if req.Method == http.MethodPost {
		l.post(resp, req)
	} else if req.Method == http.MethodGet {
		l.get(resp, req)
	} else {
		resp.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Post handles accepting new logs from the frontend.
func (l handlerImpl) post(resp http.ResponseWriter, req *http.Request) {
	// If we panic unpacking the contents, we want to return an HTTP error for a bad request.
	defer recoverFromBuffPanic(resp)

	// This will panic if the body is too large, hence the above panic handler.
	buff := new(bytes.Buffer)
	_, readErr := buff.ReadFrom(req.Body)
	if readErr != nil {
		log.Error(readErr)
	}
	closeErr := req.Body.Close()
	if closeErr != nil {
		log.Error(closeErr)
	}
	if readErr != nil || closeErr != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := l.storage.AddLog(buff.String()); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp.WriteHeader(http.StatusAccepted)
}

// Get returns all logs currently stored as a zip file.
func (l handlerImpl) get(resp http.ResponseWriter, req *http.Request) {
	// Load the logs from the db.
	logs, err := l.storage.GetLogs()
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get a new file to write to.
	compressor, err := l.compressorProvider()
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Write any logs found into the zip file. Record if we ever succeed or fail for error return.
	anyFailures := false
	anySuccesses := false
	// Each log will be a JSON object. For convenience, we wrap it in "[]" so that
	// it is readable as a JSON array.
	for i, alog := range logs {
		sb := strings.Builder{}
		if i == 0 {
			_, _ = sb.WriteString("[")
		}
		_, _ = sb.WriteString(alog)
		if i == len(logs)-1 {
			_, _ = sb.WriteString("]")
		} else {
			_, _ = sb.WriteString(",\n")
		}
		if _, err = compressor.Write([]byte(sb.String())); err != nil {
			log.Error(err)
			anyFailures = true
		} else {
			anySuccesses = true
		}
	}

	// Finish zip file and check for any bad states.
	err = compressor.Close()
	if err != nil {
		log.Warnf("Couldn't close zip writer: %s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	// If we only had write failures, return internal error.
	if anyFailures && !anySuccesses {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	// If we didn't have any content to return, then return no-content status.
	if !anySuccesses && !anyFailures {
		resp.WriteHeader(http.StatusNoContent)
		return
	}

	// If we succeeded adding some data, but failed with other, lets still write the zip out and just
	// indicate with the HTTP code that we only have part of the desired content.
	if anyFailures && anySuccesses {
		resp.WriteHeader(http.StatusPartialContent)
	}
	// If we only had successes, then YAY!
	if anySuccesses && !anyFailures {
		resp.WriteHeader(http.StatusOK)
	}

	resp.Header().Add("Content-Disposition", `attachment; filename="logs.zip"`)
	_, _ = resp.Write(compressor.Bytes())
}

func recoverFromBuffPanic(w http.ResponseWriter) {
	if r := recover(); r != nil {
		log.Error(r)
		w.WriteHeader(http.StatusBadRequest)
	}
}

// Compressor provides the interface for constructing a compressed view of the logs and supplying the bytes
// to be written to the output http.ResponseWriter. The expectation is that the Writer is used, then the
// Closer, then the BytesProvider will be able to supply the compressed data.
// In this case we are using zip compression.
//go:generate mockgen-wrapper Compressor
type Compressor interface {
	// Writing adds bytes to be compressed.
	io.Writer
	// Closing compresses the data.
	io.Closer
	// Bytes provides the compressed data after closing/compression.
	Bytes() []byte
}

// Return a zip based compressor.
func getCompressor() (Compressor, error) {
	buf := new(bytes.Buffer)
	closer := zip.NewWriter(buf)
	writer, err := closer.Create("logs.json")
	if err != nil {
		return nil, err
	}

	return &compressorImpl{
		Writer: writer,
		Closer: closer,
		buf:    buf,
	}, nil
}

// Simple compressor structure to wrap zip compression.
type compressorImpl struct {
	io.Writer
	io.Closer
	buf *bytes.Buffer
}

// Bytes just returns the bytes in the buffer storing the compressed data.
func (c *compressorImpl) Bytes() []byte {
	return c.buf.Bytes()
}
