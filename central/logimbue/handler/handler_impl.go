package handler

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/stackrox/stackrox/central/logimbue/store"
)

type handlerImpl struct {
	storage store.Store
}

// ServeHTTP adds or retrieves logs from the backend
func (l handlerImpl) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		l.post(resp, req)
	} else {
		message := fmt.Sprintf("method %s is not allowed for this endpoint", req.Method)
		http.Error(resp, message, http.StatusMethodNotAllowed)
	}
}

// post handles accepting new logs from the frontend.
func (l handlerImpl) post(resp http.ResponseWriter, req *http.Request) {
	// If we panic unpacking the contents, we want to return an HTTP error for a bad request.
	panicked := true
	defer func() {
		if r := recover(); r != nil || panicked {
			log.Error(r)
			resp.WriteHeader(http.StatusBadRequest)
		}
	}()

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
	panicked = false // from here on, any panic is no longer due to the request body
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
