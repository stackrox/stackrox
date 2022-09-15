package util

import (
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// DoHTTPRequestAndCheck200 does an http request to the provided path in Central,
// and passes through the remaining params. It checks that the returned status code is 200, and returns an error if it is not.
// The caller receives the http response object, which it is the caller's responsibility to close.
func DoHTTPRequestAndCheck200(env environment.Environment, path string, timeout time.Duration, method string, body io.Reader) (*http.Response, error) {
	client, err := env.HTTPClient(timeout)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining HTTP client")
	}

	resp, err := client.DoReqAndVerifyStatusCode(path, method, http.StatusOK, body)
	return resp, errors.Wrap(err, "making HTTP request")
}
