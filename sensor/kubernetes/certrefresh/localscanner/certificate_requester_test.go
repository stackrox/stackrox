package localscanner

import (
	"context"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrequester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

const (
	numConcurrentRequests = 10
)

var (
	testTimeout = time.Second
)

func TestCertificateRequesterRequestFailureIfStopped(t *testing.T) {
	testCases := map[string]struct {
		startRequester bool
	}{
		"requester not started":            {false},
		"requester stopped before request": {true},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			f := newFixture(0)
			defer f.tearDown()
			if tc.startRequester {
				f.requester.Start()
				f.requester.Stop()
			}

			certs, requestErr := f.requester.RequestCertificates(f.ctx)
			assert.Nil(t, certs)
			assert.Equal(t, ErrCertificateRequesterStopped, requestErr)
		})
	}
}

func TestCertificateRequesterRequestCancellation(t *testing.T) {
	f := newFixture(0)
	f.requester.Start()
	defer f.tearDown()

	f.cancelCtx()
	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.Canceled, requestErr)
}

func TestCertificateRequesterRequestSuccess(t *testing.T) {
	f := newFixture(0)
	f.requester.Start()
	defer f.tearDown()

	go f.respondRequest(t, 0, nil)

	response, err := f.requester.RequestCertificates(f.ctx)
	assert.NoError(t, err)
	assert.Equal(t, f.interceptedRequestID.Load(), response.RequestId)
}

func TestCertificateRequesterResponsesWithUnknownIDAreIgnored(t *testing.T) {
	f := newFixture(100 * time.Millisecond)
	f.requester.Start()
	defer f.tearDown()

	// Request with different request ID should be ignored.
	go f.respondRequest(t, 0, &central.IssueLocalScannerCertsResponse{RequestId: "UNKNOWN"})

	certs, requestErr := f.requester.RequestCertificates(f.ctx)
	assert.Nil(t, certs)
	assert.Equal(t, context.DeadlineExceeded, requestErr)
}

func TestCertificateRequesterRequestConcurrentRequestDoNotInterfere(t *testing.T) {
	testCases := map[string]struct {
		responseDelayFunc func(requestIndex int) (responseDelay time.Duration)
	}{
		"decreasing response delay": {func(requestIndex int) (responseDelay time.Duration) {
			// responses are responded increasingly faster, so always out of order.
			return time.Duration(numConcurrentRequests-(requestIndex+1)) * 10 * time.Millisecond
		}},
		"random response delay": {func(requestIndex int) (responseDelay time.Duration) {
			// randomly out of order responses.
			return time.Duration(rand.Intn(100)) * time.Millisecond
		}},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			f := newFixture(0)
			f.requester.Start()
			defer f.tearDown()
			waitGroup := concurrency.NewWaitGroup(numConcurrentRequests)

			for i := 0; i < numConcurrentRequests; i++ {
				i := i
				responseDelay := tc.responseDelayFunc(i)
				go f.respondRequest(t, responseDelay, nil)
				go func() {
					defer waitGroup.Add(-1)
					_, err := f.requester.RequestCertificates(f.ctx)
					assert.NoError(t, err)
				}()
			}
			ok := concurrency.WaitWithTimeout(&waitGroup, time.Duration(numConcurrentRequests)*testTimeout)
			require.True(t, ok)
		})
	}
}

func TestConvertToIssueCertsResponse(t *testing.T) {
	errorMessage := "error message"
	certificatesSet := &storage.TypedServiceCertificateSet{
		CaPem: []byte("ca_cert_pem"),
		ServiceCerts: []*storage.TypedServiceCertificate{
			{
				ServiceType: storage.ServiceType_SCANNER_SERVICE,
				Cert: &storage.ServiceCertificate{
					CertPem: []byte("scanner_cert_pem"),
					KeyPem:  []byte("scanner_key_pem"),
				},
			},
			{
				ServiceType: storage.ServiceType_SENSOR_SERVICE,
				Cert: &storage.ServiceCertificate{
					CertPem: []byte("sensor_cert_pem"),
					KeyPem:  []byte("sensor_key_pem"),
				},
			},
		},
	}

	tests := []struct {
		name           string
		input          *central.IssueLocalScannerCertsResponse
		expectedResult *certrequester.IssueCertsResponse
	}{
		{
			name:           "Nil input",
			input:          nil,
			expectedResult: nil,
		},
		{
			name: "Response with error",
			input: &central.IssueLocalScannerCertsResponse{
				RequestId: "12345",
				Response: &central.IssueLocalScannerCertsResponse_Error{
					Error: &central.LocalScannerCertsIssueError{
						Message: errorMessage,
					},
				},
			},
			expectedResult: &certrequester.IssueCertsResponse{
				RequestId:    "12345",
				ErrorMessage: &errorMessage,
				Certificates: nil,
			},
		},
		{
			name: "Response with certificates",
			input: &central.IssueLocalScannerCertsResponse{
				RequestId: "67890",
				Response: &central.IssueLocalScannerCertsResponse_Certificates{
					Certificates: certificatesSet,
				},
			},
			expectedResult: &certrequester.IssueCertsResponse{
				RequestId:    "67890",
				ErrorMessage: nil,
				Certificates: certificatesSet,
			},
		},
		{
			name:           "Empty response",
			input:          &central.IssueLocalScannerCertsResponse{},
			expectedResult: &certrequester.IssueCertsResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToIssueCertsResponse(tt.input)

			if tt.expectedResult == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expectedResult.RequestId, result.RequestId)
				assert.Equal(t, tt.expectedResult.ErrorMessage, result.ErrorMessage)
				// Must use proto.Equal for the Certificates field
				assert.True(t, proto.Equal(tt.expectedResult.Certificates, result.Certificates), "Certificates should match")
			}
		})
	}
}

type certificateRequesterFixture struct {
	sendC                chan *message.ExpiringMessage
	receiveC             chan *central.IssueLocalScannerCertsResponse
	requester            certrequester.CertificateRequester
	interceptedRequestID *atomic.Value
	ctx                  context.Context
	cancelCtx            context.CancelFunc
}

// newFixture creates a new test fixture that uses `timeout` as context timeout if `timeout` is
// not 0, and `testTimeout` otherwise.
func newFixture(timeout time.Duration) *certificateRequesterFixture {
	sendC := make(chan *message.ExpiringMessage)
	receiveC := make(chan *central.IssueLocalScannerCertsResponse)
	requester := NewCertificateRequester(sendC, receiveC)
	var interceptedRequestID atomic.Value
	if timeout == 0 {
		timeout = testTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &certificateRequesterFixture{
		sendC:                sendC,
		receiveC:             receiveC,
		requester:            requester,
		ctx:                  ctx,
		cancelCtx:            cancel,
		interceptedRequestID: &interceptedRequestID,
	}
}

func (f *certificateRequesterFixture) tearDown() {
	f.cancelCtx()
	f.requester.Stop()
}

// respondRequest reads a request from `f.sendC` and responds with `responseOverwrite` if not nil, or with
// a response with the same ID as the request otherwise. If `responseDelay` is greater than 0 then this function
// waits for that time before sending the response.
// Before sending the response, it stores in `f.interceptedRequestID` the request ID for the requests read from `f.sendC`.
func (f *certificateRequesterFixture) respondRequest(t *testing.T, responseDelay time.Duration, responseOverwrite *central.IssueLocalScannerCertsResponse) {
	select {
	case <-f.ctx.Done():
	case request := <-f.sendC:
		interceptedRequestID := request.GetIssueLocalScannerCertsRequest().GetRequestId()
		assert.NotEmpty(t, interceptedRequestID)
		var response *central.IssueLocalScannerCertsResponse
		if responseOverwrite != nil {
			response = responseOverwrite
		} else {
			response = &central.IssueLocalScannerCertsResponse{RequestId: interceptedRequestID}
		}
		f.interceptedRequestID.Store(response.GetRequestId())
		if responseDelay > 0 {
			select {
			case <-f.ctx.Done():
				return
			case <-time.After(responseDelay):
			}
		}
		select {
		case <-f.ctx.Done():
		case f.receiveC <- response:
		}
	}
}
