package azure

import (
	"net/http"
	"time"
)

const (
	timeout = 5 * time.Second
)

var httpClient = &http.Client{
	Timeout: timeout,
}
