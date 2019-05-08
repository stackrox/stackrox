package gcp

import (
	"net/http"
	"time"
)

const (
	timeout = 60 * time.Second
)

var httpClient = &http.Client{
	Timeout: timeout,
}
