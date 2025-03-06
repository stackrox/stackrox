package httputil

import (
	"crypto/tls"
	"net/http"
)

// InsecureSkipTLSVerifyHeader specifies the custom header used to
// identify when to skip TLS verification.
const InsecureSkipTLSVerifyHeader = `StackRox-Insecure-Skip-TLS-Verify`

var _ http.RoundTripper = (*insecureCapableTransport)(nil)

// insecureCapableTransport is a http.RoundTripper
// which supports communication with servers with untrusted certificates
// via presence of the header InsecureSkipTLSVerifyHeader.
type insecureCapableTransport struct {
	transport         http.RoundTripper
	insecureTransport http.RoundTripper
}

// NewInsecureCapableTransport creates a http.RoundTripper based on the given transport.
//
// The given transport is used for all requests unless the request has the InsecureSkipTLSVerifyHeader
// header. In that case, a copy of the given transport which skips client-side TLS certificate verification
// is used.
func NewInsecureCapableTransport(transport *http.Transport) http.RoundTripper {
	insecure := transport.Clone()
	if insecure.TLSClientConfig == nil {
		insecure.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	} else {
		insecure.TLSClientConfig.InsecureSkipVerify = true
	}
	return &insecureCapableTransport{
		transport:         transport,
		insecureTransport: insecure,
	}
}

func (t *insecureCapableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get(InsecureSkipTLSVerifyHeader) == "true" {
		req.Header.Del(InsecureSkipTLSVerifyHeader)
		return t.insecureTransport.RoundTrip(req)
	}
	return t.transport.RoundTrip(req)
}
