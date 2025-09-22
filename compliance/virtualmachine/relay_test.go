package virtualmachine

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestVMRelay(t *testing.T) {
	suite.Run(t, new(relayTestSuite))
}

type relayTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *relayTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *relayTestSuite) TestVsockConnectionHandlerInjectsVsockCID() {
	conn := s.defaultVsockConn().withVsockCID(42)
	client := newMockSensorClient()

	err := handleVsockConnection(s.ctx, conn, client)
	require.NoError(s.T(), err)

	s.True(conn.closed, "connection should be closed after handling")

	s.Equal("42", client.capturedRequests[0].IndexReport.VsockCid)
}

func (s *relayTestSuite) TestVsockConnectionHandlerRejectsMalformedData() {
	conn := s.defaultVsockConn().withData([]byte("malformed-data"))
	client := newMockSensorClient()

	err := handleVsockConnection(s.ctx, conn, client)
	require.Error(s.T(), err)

	s.True(conn.closed, "connection should be closed after handling")
}

func (s *relayTestSuite) defaultVsockConn() *mockVsockConn {
	c := newMockVsockConn().withVsockCID(1234)
	c, err := c.withIndexReport(&v1.IndexReport{})
	s.NoError(err)
	return c
}
