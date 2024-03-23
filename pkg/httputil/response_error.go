package httputil

import (
	"fmt"
	"io"
	"net/http"
)

// ResponseToError converts a response to an HTTP request to an error (or nil, if it is a 2xx response code).
// If the response indicates an error, the response body is read (but not closed).
func ResponseToError(resp *http.Response) HTTPError {
	if Is2xxStatusCode(resp.StatusCode) {
		return nil
	}

	var errMsg string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errMsg = fmt.Sprintf("error reading response body: %v", err)
	} else {
		errMsg = string(body)
	}
	return httpError{
		code:    resp.StatusCode,
		message: fmt.Sprintf("received response code %s, but expected 2xx; error message: %s", resp.Status, errMsg),
	}
}
