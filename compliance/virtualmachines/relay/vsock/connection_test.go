package vsock

import (
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stretchr/testify/suite"
)

func TestVsockConnection(t *testing.T) {
	suite.Run(t, new(connectionTestSuite))
}

type connectionTestSuite struct {
	suite.Suite
}

func (s *connectionTestSuite) TestExtractVsockCIDFromConnection() {

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

func (s *connectionTestSuite) TestReadFromConn() {
	data := []byte("Hello, world!")

	cases := map[string]struct {
		delay       time.Duration
		maxSize     int
		readTimeout time.Duration
		shouldError bool
	}{
		"data smaller than limit succeeds": {
			maxSize:     2 * len(data),
			readTimeout: 10 * time.Second,
			shouldError: false,
		},
		"data of equal size as limit succeeds": {
			maxSize:     len(data),
			readTimeout: 10 * time.Second,
			shouldError: false,
		},
		"data larger than limit fails": {
			maxSize:     len(data) - 1,
			readTimeout: 10 * time.Second,
			shouldError: true,
		},
		"delay longer than timeout fails": {
			maxSize:     len(data),
			delay:       1 * time.Second,
			readTimeout: 100 * time.Millisecond,
			shouldError: true,
		},
		"delay shorter than timeout succeeds": {
			maxSize:     len(data),
			delay:       100 * time.Millisecond,
			readTimeout: 1 * time.Second,
			shouldError: false,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			conn := newMockVsockConn().withData(data).withDelay(c.delay)

			readData, err := ReadFromConn(conn, c.maxSize, c.readTimeout, 12345)
			if c.shouldError {
				s.Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(data, readData)
			}
		})
	}
}

type mockVsockConn struct {
	closed       bool
	data         []byte
	delay        time.Duration
	readErr      error
	remoteAddr   net.Addr
	readDeadline time.Time
}

func newMockVsockConn() *mockVsockConn {
	return &mockVsockConn{}
}

func (c *mockVsockConn) withVsockCID(vsockCID uint32) *mockVsockConn {
	c.remoteAddr = &vsock.Addr{ContextID: vsockCID}
	return c
}

func (c *mockVsockConn) withData(data []byte) *mockVsockConn {
	c.data = data
	return c
}

func (c *mockVsockConn) withDelay(delay time.Duration) *mockVsockConn {
	c.delay = delay
	return c
}

func (c *mockVsockConn) Read(b []byte) (n int, err error) {
	time.Sleep(c.delay)
	if !c.readDeadline.IsZero() && time.Now().After(c.readDeadline) {
		return 0, os.ErrDeadlineExceeded
	}
	if c.readErr != nil {
		return 0, c.readErr
	}
	n = copy(b, c.data)
	if n == len(c.data) {
		return n, io.EOF
	}
	return n, nil
}

func (c *mockVsockConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *mockVsockConn) Close() error {
	c.closed = true
	return nil
}

func (c *mockVsockConn) Write([]byte) (int, error)   { return 0, nil }
func (c *mockVsockConn) LocalAddr() net.Addr         { return nil }
func (c *mockVsockConn) SetDeadline(time.Time) error { return nil }
func (c *mockVsockConn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}
func (c *mockVsockConn) SetWriteDeadline(time.Time) error { return nil }
