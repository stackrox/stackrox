package proxy

import (
	"net"
	"net/http"
	"time"
)

var (
	defaultDialer = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
)

func copyDefaultTransport() *http.Transport {
	trans, _ := http.DefaultTransport.(*http.Transport)
	if trans != nil {
		trans = trans.Clone()
	} else {
		// fallback copied from go http/transport.go, circa 1.13.1.
		trans = &http.Transport{
			DialContext:           defaultDialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}
	return trans
}
