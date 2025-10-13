package certrefresh

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// Response represents the response to a certificate request. It contains a set of certificates or an error.
type Response struct {
	RequestId    string
	ErrorMessage *string
	Certificates *storage.TypedServiceCertificateSet
}

// NewResponseFromSecuredClusterCerts creates a certificates.Response from a
// protobuf central.IssueSecuredClusterCertsResponse message
func NewResponseFromSecuredClusterCerts(response *central.IssueSecuredClusterCertsResponse) *Response {
	if response == nil {
		return nil
	}

	res := &Response{
		RequestId: response.GetRequestId(),
	}

	if response.GetError() != nil {
		errMsg := response.GetError().GetMessage()
		res.ErrorMessage = &errMsg
	} else {
		res.Certificates = response.GetCertificates()
	}

	return res
}
