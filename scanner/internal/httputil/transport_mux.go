package httputil

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/urlfmt"
)

var (
	// defaultDialer is essentially copied from http.DefaultTransport from go1.20.10.
	defaultDialer = net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
)

type options struct {
	denyCentral bool
	denySensor  bool

	// These are here for testing purposes.
	centralTransport http.RoundTripper
	sensorTransport  http.RoundTripper
}

// TransportOption configures options for HTTP transport.
type TransportOption func(o *options)

// WithDenyStackRoxServices configures whether the transport should deny traffic to all StackRox services.
//
// Default: false
func WithDenyStackRoxServices(deny bool) TransportOption {
	return func(o *options) {
		WithDenyCentral(deny)(o)
		WithDenySensor(deny)(o)
	}
}

// WithDenyCentral configures whether the transport should deny traffic to Central.
//
// Default: false
func WithDenyCentral(deny bool) TransportOption {
	return func(o *options) {
		o.denyCentral = deny
	}
}

// WithDenySensor configures whether the transport should deny traffic to Sensor.
//
// Default: false
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

	return transportMux(defaultTransport, o)
}

func transportMux(defaultTransport http.RoundTripper, o options) (http.RoundTripper, error) {
	centralTransport := DenyTransport
	if !o.denyCentral {
		centralTransport = o.centralTransport
		if centralTransport == nil {
			var err error
			centralTransport, err = roxTransport(mtls.CentralSubject)
			if err != nil {
				return nil, fmt.Errorf("creating Central TLS config: %w", err)
			}
		}
	}

	sensorTransport := DenyTransport
	if !o.denySensor {
		sensorTransport = o.sensorTransport
		if sensorTransport == nil {
			var err error
			sensorTransport, err = roxTransport(mtls.SensorSubject)
			if err != nil {
				return nil, fmt.Errorf("creating Sensor TLS config: %w", err)
			}
		}
	}

	// Set the defaults here instead of in global variables for testing purposes.
	centralHostname, err := hostname(env.CentralEndpoint.Setting(), fmt.Sprintf("central.%s.svc", env.Namespace.Setting()))
	if err != nil {
		return nil, err
	}
	sensorHostname, err := hostname(env.SensorEndpoint.Setting(), fmt.Sprintf("sensor.%s.svc", env.Namespace.Setting()))
	if err != nil {
		return nil, err
	}

	return httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Hostname() {
		case centralHostname:
			return centralTransport.RoundTrip(req)
		case sensorHostname:
			return sensorTransport.RoundTrip(req)
		default:
			return defaultTransport.RoundTrip(req)
		}
	}), nil
}

func hostname(endpoint, defaultHostname string) (string, error) {
	if endpoint == "" {
		return defaultHostname, nil
	}
	// This will prepend the https:// schema to the URL allowing us to get an accurate
	// value for the Hostname call below.
	endpoint = urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parsing URL %q: %w", endpoint, err)
	}
	return parsedURL.Hostname(), nil
}

func roxTransport(subject mtls.Subject) (http.RoundTripper, error) {
	tlsConfig, err := clientconn.TLSConfig(subject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		return nil, fmt.Errorf("configuring TLS: %w", err)
	}
	// TODO(ROX-21861): clientconn.TLSConfig prefers HTTP/2 traffic over HTTP/1.1.
	// At the moment, we are receiving status code 421 from StackRox services,
	// so clear NextProtos to ensure we only use HTTP/1.x.
	tlsConfig.NextProtos = nil

	return &http.Transport{
		Proxy:           proxy.FromConfig(),
		TLSClientConfig: tlsConfig,
		// TODO(ROX-21861): When enabled, we receive status code 421.
		// For now, disallow all HTTP/2 traffic to StackRox services.
		ForceAttemptHTTP2: false,

		// The rest are (more-or-less) copied from http.DefaultTransport as of go1.24.2.
		DialContext:           defaultDialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}, nil
}
