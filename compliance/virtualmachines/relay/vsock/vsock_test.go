package vsock

import (
	"net"
	"testing"

	relaytest "github.com/stackrox/rox/compliance/virtualmachines/relay/testutils"
	"github.com/stretchr/testify/suite"
)

func TestVsock(t *testing.T) {
	suite.Run(t, new(vsockTestSuite))
}

type vsockTestSuite struct {
	suite.Suite
}

func (s *vsockTestSuite) TestExtractVsockCIDFromConnection() {

	connWrongAddrType := relaytest.NewMockVsockConn(s.T()).WithVsockCID(1234)
	connWrongAddrType.SetRemoteAddr(&net.TCPAddr{})

	cases := map[string]struct {
		conn             net.Conn
		shouldError      bool
		expectedVsockCID uint32
	}{
		"wrong type fails": {
			conn:             connWrongAddrType,
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"reserved vsock CID fails": {
			conn:             relaytest.NewMockVsockConn(s.T()).WithVsockCID(2),
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"valid vsock CID succeeds": {
			conn:             relaytest.NewMockVsockConn(s.T()).WithVsockCID(42),
			shouldError:      false,
			expectedVsockCID: 42,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			vsockCID, err := ExtractVsockCIDFromConnection(c.conn)
			if c.shouldError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(c.expectedVsockCID, vsockCID)
			}
		})
	}
}
