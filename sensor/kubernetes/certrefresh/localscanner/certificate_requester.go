package localscanner

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certificates"
)

// NewCertificateRequester creates a new certificate requester that communicates through
// the specified channels and initializes a new request ID for reach request.
// To use it call Start, and then make requests with RequestCertificates, concurrent requests are supported.
// This assumes that the returned certificate requester is the only consumer of `receiveC`.
func NewCertificateRequester(sendC chan<- *message.ExpiringMessage,
	receiveC <-chan *central.IssueLocalScannerCertsResponse) certificates.Requester {
	return certificates.NewRequester[
		*central.IssueLocalScannerCertsRequest,
		*central.IssueLocalScannerCertsResponse,
	](
		sendC,
		receiveC,
		&certificates.LocalScannerMessageFactory{},
		&certificates.LocalScannerResponseFactory{},
	)
}
