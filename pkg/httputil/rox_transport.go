package httputil

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
)

// RoxTransportOptions represents client options for reaching out to Rox-related services.
type RoxTransportOptions struct {
	disableCompression bool
}

func RoxRoundTripInterceptor(subject mtls.Subject, o RoxTransportOptions) (RoundTripInterceptor, error) {
	transport, err := RoxTransport(subject, o)
	if err != nil {
		return nil, err
	}

	return func(req *http.Request, roundTrip RoundTripperFunc) (*http.Response, error) {
		if strings.Contains(req.URL.Host, host(subject)) {
			return transport.RoundTrip(req)
		}
		return roundTrip(req)
	}, nil
}

func RoxTransport(subject mtls.Subject, o RoxTransportOptions) (http.RoundTripper, error) {
	tlsConfig, err := clientconn.TLSConfig(subject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		return nil, err
	}

	return &http.Transport{
			Proxy:              proxy.FromConfig(),
			TLSClientConfig:    tlsConfig,
			DisableCompression: o.disableCompression,
	}, nil
}

func host(subject mtls.Subject) string {
	switch subject.ServiceType {
	case storage.ServiceType_CENTRAL_SERVICE:
		return "central"
	case storage.ServiceType_SENSOR_SERVICE:
		return "sensor"
	default:
		utils.Should(errors.Errorf("unexpected service type %v", subject.ServiceType))
	}
	return ""
}
