package localscanner

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
)

var (
	_ certificateRequest = (*certRequestSyncImpl)(nil)
)

// certificateRequest request a new set of local scanner certificates to central.
type certificateRequest interface {
	requestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)
}

type certRequestSyncImpl struct {
	requestID      string
	msgFromSensorC msgFromSensorC
	msgToSensorC   msgToSensorC
}

func (i *certRequestSyncImpl) requestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error) {
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
		log.Debugf("request to issue local Scanner certificates sent to Central successfully: %v", msg)
	}

	var response *central.IssueLocalScannerCertsResponse
	for response == nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case newResponse := <-i.msgToSensorC:
			if newResponse.GetRequestId() != i.requestID {
				log.Debugf("request id %q does not match %q, skipping request", response.GetRequestId(),
					i.requestID)
			} else {
				response = newResponse
			}
		}
	}

	return response, nil
}
