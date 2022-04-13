package azure

import (
	"net/http"
	"time"

	"github.com/stackrox/stackrox/pkg/httputil/proxy"
)

const (
	timeout = 5 * time.Second
)

var (
	metadataHTTPClient = &http.Client{
		Timeout:   timeout,
		Transport: proxy.Without(),
	}
	certificateHTTPClient = &http.Client{
		Transport: proxy.RoundTripper(),
		Timeout:   timeout,
	}
)
