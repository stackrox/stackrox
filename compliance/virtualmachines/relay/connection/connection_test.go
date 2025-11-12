package connection

import (
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func TestConnection(t *testing.T) {
	suite.Run(t, new(connectionTestSuite))
}

type connectionTestSuite struct {
	suite.Suite
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
			conn := newMockConn().withData(data).withDelay(c.delay)

			readData, err := ReadFromConn(conn, c.maxSize, c.readTimeout)
			if c.shouldError {
				s.Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(data, readData)
			}
		})
	}
}

type mockConn struct {
	closed       bool
	data         []byte
	delay        time.Duration
	readErr      error
	remoteAddr   net.Addr
	readDeadline time.Time
}

func newMockConn() *mockConn {
	return &mockConn{}
}

func (c *mockConn) withData(data []byte) *mockConn {
	c.data = data
	return c
}

func (c *mockConn) withDelay(delay time.Duration) *mockConn {
	c.delay = delay
	return c
}

func (c *mockConn) Read(b []byte) (n int, err error) {
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

func (c *mockConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *mockConn) Close() error {
	c.closed = true
	return nil
}

func (c *mockConn) Write([]byte) (int, error)   { return 0, nil }
func (c *mockConn) LocalAddr() net.Addr         { return nil }
func (c *mockConn) SetDeadline(time.Time) error { return nil }
func (c *mockConn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}
func (c *mockConn) SetWriteDeadline(time.Time) error { return nil }
