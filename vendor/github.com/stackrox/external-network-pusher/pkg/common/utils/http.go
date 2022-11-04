package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const httpGetTimeout = 60 * time.Second

// HTTPGet returns the body of the HTTP GET response
func HTTPGet(url string) ([]byte, error) {
	log.Printf("Getting from URL: %s...", url)

	client := &http.Client{
		Timeout: httpGetTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non 200 status code. Code: %d, error: %v", resp.StatusCode, err)
	}

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed while trying to copy response data")
	}
	return bodyData, nil
}
