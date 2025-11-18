package handler

import (
	"context"
	"strconv"
	"testing"
	"time"

	relaytest "github.com/stackrox/rox/compliance/virtualmachines/relay/testutils"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(handlerTestSuite))
}

type handlerTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *handlerTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *handlerTestSuite) TestParseIndexReport() {
	data := []byte("malformed-data")
	parsedIndexReport, err := parseIndexReport(data)
	s.Require().Error(err)
	s.Require().Nil(parsedIndexReport)

	validIndexReport := &v1.IndexReport{VsockCid: "42"}
	data, err = proto.Marshal(validIndexReport)
	s.Require().NoError(err)
	parsedIndexReport, err = parseIndexReport(data)
	s.Require().NoError(err)
	s.Require().True(proto.Equal(validIndexReport, parsedIndexReport))
}

func (s *handlerTestSuite) TestValidateVsockCID() {
	// Reported CID is 42
	indexReport := v1.IndexReport{VsockCid: "42"}

	// Real (connection) CID is 99 - does not match, should return error
	connVsockCID := uint32(99)
	err := validateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().Error(err)

	// Real (connection) CID is 42 - matches, should return nil
	connVsockCID = uint32(42)
	err = validateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().NoError(err)
}

func (s *handlerTestSuite) TestHandle_RejectsMismatchingVsockCID() {
	cases := map[string]struct {
		indexReportVsockCID int
		connVsockCID        int
		shouldError         bool
	}{
		"mismatching vsock CID fails": {
			indexReportVsockCID: 42,
			connVsockCID:        99,
			shouldError:         true,
		},
		"matching vsock CID succeeds": {
			indexReportVsockCID: 42,
			connVsockCID:        42,
			shouldError:         false,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			indexReport := &v1.IndexReport{VsockCid: strconv.Itoa(c.indexReportVsockCID)}
			conn, err := relaytest.NewMockVsockConn().WithVsockCID(uint32(c.connVsockCID)).WithIndexReport(indexReport)
			s.Require().NoError(err)
			client := relaytest.NewMockSensorClient()

			handler := New(client)
			err = handler.Handle(s.ctx, conn)
			if c.shouldError {
				s.Require().Error(err)
				s.Contains(err.Error(), "mismatch")
				s.Empty(client.CapturedRequests())
			} else {
				s.Require().NoError(err)
				s.Len(client.CapturedRequests(), 1)
			}
		})
	}
}

func (s *handlerTestSuite) TestHandle_RejectsMalformedData() {
	conn := relaytest.NewMockVsockConn().WithVsockCID(1234).WithData([]byte("malformed-data"))
	client := relaytest.NewMockSensorClient()
	handler := New(client)

	err := handler.Handle(s.ctx, conn)
	s.Error(err)
}

func (s *handlerTestSuite) TestHandle_HandlesContextCancellation() {
	indexReport := &v1.IndexReport{VsockCid: "1234"}
	conn, err := relaytest.NewMockVsockConn().WithVsockCID(1234).WithIndexReport(indexReport)
	s.Require().NoError(err)

	// Set up a sensor client that only returns after 500 ms
	client := relaytest.NewMockSensorClient().WithDelay(500 * time.Millisecond)

	// Set up a context that will be canceled after 100 ms
	cancellableCtx, cancel := context.WithCancel(s.ctx)
	handler := New(client)
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// When the connection is handled, sending the index report to sensor will hang for 500 ms.
	// The context will be canceled after 100 ms, that is, while the sensor response is awaited.
	// Therefore we expect a "context canceled" error
	err = handler.Handle(cancellableCtx, conn)
	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
}
