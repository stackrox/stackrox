// Package httputil defines utility HTTP functions built specifically for Scanner's use cases.
//
// It is possible other StackRox components may benefit from these functions,
// but that will be considered at a future time.
package httputil

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	namespace = env.Namespace.Setting()

	centralHost = fmt.Sprintf("central.%s.svc", namespace)
	sensorHost  = fmt.Sprintf("sensor.%s.svc", namespace)

	// defaultDialer is essentially copied from http.DefaultTransport from go1.20.10.
	defaultDialer = net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
)

type options struct {
	denySensor bool
}

// TransportOption configures options for HTTP transport.
type TransportOption func(o *options)

// WithDenySensor configures whether the transport should deny traffic to Sensor.
func WithDenySensor(deny bool) TransportOption {
	return func(o *options) {
		o.denySensor = deny
	}
}

// TransportMux returns a http.RoundTripper which multiplexes the given default http.RoundTripper
// with ones which support mTLS with StackRox services.
//
// It is assumed the StackRox services are in the same Kubernetes namespace.
//
// Only Central and Sensor are supported StackRox services at this time.
func TransportMux(defaultTransport http.RoundTripper, opts ...TransportOption) (http.RoundTripper, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	centralTransport, err := roxTransport(mtls.CentralSubject, o)
	if err != nil {
		return nil, fmt.Errorf("creating Central TLS config: %w", err)
	}

	sensorTransport := DenyTransport
	if !o.denySensor {
		sensorTransport, err = roxTransport(mtls.SensorSubject, o)
		if err != nil {
			return nil, fmt.Errorf("creating Sensor TLS config: %w", err)
		}
	}

	return httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case centralHost:
			return centralTransport.RoundTrip(req)
		case sensorHost:
			return sensorTransport.RoundTrip(req)
		default:
			return defaultTransport.RoundTrip(req)
		}
	}), nil
}

func roxTransport(subject mtls.Subject, o options) (http.RoundTripper, error) {
	tlsConfig, err := clientconn.TLSConfig(subject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		return nil, fmt.Errorf("configuring TLS: %w", err)
	}
	// clientConn.TLSConfig currently prefers gRPC, which Central hosts on the same port as HTTP.
	// Clear NextProtos so we don't accidentally prefer gRPC traffic over HTTP.
	// NextProtos will be repopulated via http2.ConfigureTransport.
	tlsConfig.NextProtos = nil

	transport := &http.Transport{
		Proxy:           proxy.FromConfig(),
		TLSClientConfig: tlsConfig,
		ForceAttemptHTTP2: true,

		// The rest are (more-or-less) copied from http.DefaultTransport as of go1.20.10.
		DialContext:           defaultDialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return transport, nil
}
