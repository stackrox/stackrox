package vsockdialer

import (
	"errors"
	"io"
	"net"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConn is a minimal net.Conn whose Read returns a fixed error, enough to
// exercise closeCodeConn's translation without a real websocket connection.
type fakeConn struct {
	net.Conn
	readErr error
}

func (c *fakeConn) Read([]byte) (int, error) {
	return 0, c.readErr
}

func TestCloseCodeConn_Read(t *testing.T) {
	cases := map[string]struct {
		readErr   error
		wantCode  int
		wantPlain bool // true if the error should pass through unchanged
	}{
		"passes a non-close error through unchanged": {
			readErr:   errors.New("boom"),
			wantPlain: true,
		},
		"passes a plain io.EOF through unchanged": {
			readErr:   io.EOF,
			wantPlain: true,
		},
		"translates an abnormal websocket.CloseError to a structured closedError": {
			readErr:  &websocket.CloseError{Code: websocket.CloseAbnormalClosure, Text: "abnormal closure"},
			wantCode: websocket.CloseAbnormalClosure,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			conn := &closeCodeConn{Conn: &fakeConn{readErr: tc.readErr}}
			_, err := conn.Read(make([]byte, 1))

			if tc.wantPlain {
				assert.Equal(t, tc.readErr, err)
				return
			}

			var closeErr *closedError
			require.ErrorAs(t, err, &closeErr)
			code, _ := closeErr.CloseCode()
			assert.Equal(t, tc.wantCode, code)
		})
	}
}
