package gcp

import (
	"net/http"
	"time"

	"github.com/stackrox/stackrox/pkg/httputil/proxy"
)

const (
	timeout = 60 * time.Second
)

var (
	metadataHTTPClient = &http.Client{
		Timeout:   timeout,
		Transport: proxy.Without(),
	}

	certificateHTTPClient = &http.Client{
		Timeout:   timeout,
		Transport: proxy.RoundTripper(),
	}
)
