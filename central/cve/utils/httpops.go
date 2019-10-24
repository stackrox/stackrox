package utils

import (
	"net/http"
	"time"
)

var (
	client = &http.Client{
		Timeout: 60 * time.Second,
	}
)

// RunHTTPGet runs an HTTP GET request
func RunHTTPGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ReadNBytesFromResponse reads N bytes from an HTTP response
func ReadNBytesFromResponse(r *http.Response, n int) ([]byte, error) {
	buf := make([]byte, n)
	nRead, err := r.Body.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:nRead], nil
}
