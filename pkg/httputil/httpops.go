package httputil

import (
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

var (
	timeout = 60 * time.Second
)

// HTTPGet run a HTTP GET request and returns the body of response
func HTTPGet(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("Failed to close response body for HTTP GET %q: %v", url, err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to GET: %q; received status code: %d", url, resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading HTTP GET response")
	}
	return bytes, nil
}
