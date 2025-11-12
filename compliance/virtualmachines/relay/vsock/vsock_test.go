package vsock

import (
	"net"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stretchr/testify/suite"
)

func TestVsock(t *testing.T) {
	suite.Run(t, new(vsockTestSuite))
}

type vsockTestSuite struct {
	suite.Suite
}

func (s *vsockTestSuite) TestExtractVsockCIDFromConnection() {

	connWrongAddrType := newMockVsockConn().withVsockCID(1234)
	connWrongAddrType.remoteAddr = &net.TCPAddr{}

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
			conn:             newMockVsockConn().withVsockCID(2),
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"valid vsock CID succeeds": {
			conn:             newMockVsockConn().withVsockCID(42),
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

type mockVsockConn struct {
	remoteAddr net.Addr
}

func newMockVsockConn() *mockVsockConn {
	return &mockVsockConn{}
}

func (c *mockVsockConn) withVsockCID(vsockCID uint32) *mockVsockConn {
	c.remoteAddr = &vsock.Addr{ContextID: vsockCID}
	return c
}

func (c *mockVsockConn) RemoteAddr() net.Addr               { return c.remoteAddr }
func (c *mockVsockConn) Read([]byte) (int, error)           { return 0, nil }
func (c *mockVsockConn) Write([]byte) (int, error)          { return 0, nil }
func (c *mockVsockConn) Close() error                       { return nil }
func (c *mockVsockConn) LocalAddr() net.Addr                { return nil }
func (c *mockVsockConn) SetDeadline(t time.Time) error      { return nil }
func (c *mockVsockConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *mockVsockConn) SetWriteDeadline(t time.Time) error { return nil }
