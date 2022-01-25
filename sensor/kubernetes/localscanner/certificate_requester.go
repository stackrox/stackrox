package localscanner

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	_ CertificateRequester = (*certRequesterSyncImpl)(nil)
)

// CertificateRequester request a new set of local scanner certificates to central.
type CertificateRequester interface {
	RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)
}

// NewCertificateRequester creates a new CertificateRequester that communicates through
// the specified channels, and that uses a fresh request ID.
func NewCertificateRequester(msgFromSensorC chan *central.MsgFromSensor,
	msgToSensorC chan *central.IssueLocalScannerCertsResponse) CertificateRequester {
	return &certRequesterSyncImpl{
		requestID:      uuid.NewV4().String(),
		msgFromSensorC: msgFromSensorC,
		msgToSensorC:   msgToSensorC,
	}
}

type certRequesterSyncImpl struct {
	requestID      string
	msgFromSensorC chan *central.MsgFromSensor
	msgToSensorC   chan *central.IssueLocalScannerCertsResponse
}

func (i *certRequesterSyncImpl) RequestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error) {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_IssueLocalScannerCertsRequest{
			IssueLocalScannerCertsRequest: &central.IssueLocalScannerCertsRequest{
				RequestId: i.requestID,
			},
		},
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case i.msgFromSensorC <- msg:
		log.Debugf("request to issue local Scanner certificates sent to Central succesfully: %v", msg)
	}

	var response *central.IssueLocalScannerCertsResponse
	for response == nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case newResponse := <-i.msgToSensorC:
			if newResponse.GetRequestId() != i.requestID {
				log.Debugf("ignoring response with unknown request id %s", response.GetRequestId())
			} else {
				response = newResponse
			}
		}
	}

	return response, nil
}
