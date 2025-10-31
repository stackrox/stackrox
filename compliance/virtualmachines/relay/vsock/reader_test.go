package vsock

import (
	"context"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

func TestVsockReader(t *testing.T) {
	suite.Run(t, new(readerTestSuite))
}

type readerTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *readerTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *readerTestSuite) TestExtractVsockCIDFromConnection() {
	connWrongAddrType := s.defaultVsockConn()
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
			conn:             s.defaultVsockConn().withVsockCID(2),
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"valid vsock CID succeeds": {
			conn:             s.defaultVsockConn().withVsockCID(42),
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

func (s *readerTestSuite) TestParseIndexReport() {
	data := []byte("malformed-data")
	parsedIndexReport, err := ParseIndexReport(data)
	s.Require().Error(err)
	s.Require().Nil(parsedIndexReport)

	validIndexReport := &v1.IndexReport{VsockCid: "42"}
	data, err = proto.Marshal(validIndexReport)
	s.Require().NoError(err)
	parsedIndexReport, err = ParseIndexReport(data)
	s.Require().NoError(err)
	s.Require().True(proto.Equal(validIndexReport, parsedIndexReport))
}

func (s *readerTestSuite) TestReadFromConn() {
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
			conn := s.defaultVsockConn().withData(data).withDelay(c.delay)

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

func (s *readerTestSuite) TestValidateVsockCID() {
	// Reported CID is 42
	indexReport := v1.IndexReport{VsockCid: "42"}

	// Real (connection) CID is 99 - does not match, should return error
	connVsockCID := uint32(99)
	err := ValidateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().Error(err)

	// Real (connection) CID is 42 - matches, should return nil
	connVsockCID = uint32(42)
	err = ValidateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().NoError(err)
}

func (s *readerTestSuite) defaultVsockConn() *mockVsockConn {
	c := newMockVsockConn().withVsockCID(1234)
	c, err := c.withIndexReport(&v1.IndexReport{VsockCid: "1234"})
	s.Require().NoError(err)
	return c
}

// Mock vsock connection for testing
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

func (c *mockVsockConn) withIndexReport(indexReport *v1.IndexReport) (*mockVsockConn, error) {
	reportData, err := proto.Marshal(indexReport)
	c.data = reportData
	return c, err
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

func (c *mockVsockConn) Write([]byte) (int, error)             { return 0, nil }
func (c *mockVsockConn) LocalAddr() net.Addr                   { return nil }
func (c *mockVsockConn) SetDeadline(time.Time) error           { return nil }
func (c *mockVsockConn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}
func (c *mockVsockConn) SetWriteDeadline(time.Time) error { return nil }
