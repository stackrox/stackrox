package csv

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

type mockResponseWriter struct {
	header http.Header
	status int
}

func (m *mockResponseWriter) Header() (h http.Header)           { return m.header }
func (m *mockResponseWriter) Write(p []byte) (n int, err error) { return len(p), nil }
func (m *mockResponseWriter) WriteHeader(statusCode int)        { m.status = statusCode }

func TestWriteError(t *testing.T) {
	type testCase struct {
		desc         string
		err          error
		expectedCode int
	}

	testCases := []testCase{
		{
			"Known error maps to the appropriate HTTP code",
			errox.InvalidArgs,
			400,
		},
		{
			"Unknown error maps to the default HTTP code",
			errors.New("bigbadabum"),
			500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			w := &mockResponseWriter{header: http.Header{}}
			WriteError(w, tc.err)
			assert.Equal(t, tc.expectedCode, w.status)
			assert.Contains(t, w.header["Content-Type"][0], "text/plain")
		})
	}
}
