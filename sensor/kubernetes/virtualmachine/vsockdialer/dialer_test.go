package vsockdialer

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportClosedErr(t *testing.T) {
	cases := map[string]struct {
		err      error
		wantNil  bool
		wantCode int
		wantEOF  bool
	}{
		"should return nil for a non-close error": {
			err:     errors.New("boom"),
			wantNil: true,
		},
		"should classify a plain io.EOF with no structured code": {
			err:      io.EOF,
			wantCode: 0,
			wantEOF:  true,
		},
		"should preserve the close code from a websocket.CloseError": {
			err:      &websocket.CloseError{Code: websocket.CloseAbnormalClosure, Text: "abnormal closure"},
			wantCode: websocket.CloseAbnormalClosure,
		},
		"should treat a normal closure as io.EOF": {
			err:      fmt.Errorf("read: %w", &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "bye"}),
			wantCode: websocket.CloseNormalClosure,
			wantEOF:  true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := transportClosedErr(tc.err)
			if tc.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			code, _ := got.CloseCode()
			assert.Equal(t, tc.wantCode, code)
			if tc.wantEOF {
				assert.ErrorIs(t, got, io.EOF)
				return
			}
			assert.NotErrorIs(t, got, io.EOF)
		})
	}
}
