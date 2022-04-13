package aws

import (
	"net/http"
	"time"

	"github.com/stackrox/stackrox/pkg/httputil/proxy"
)

const (
	timeout = 5 * time.Second
)

var httpClient = &http.Client{
	Timeout:   timeout,
	Transport: proxy.Without(),
}
