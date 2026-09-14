package certgen

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScannerHandlerRejectsUnsupportedVersion(t *testing.T) {
	s := &serviceImpl{}
	cases := map[string]struct {
		method string
		rawURL string
		want   int
	}{
		"should reject GET": {
			method: http.MethodGet,
			rawURL: "/api/extensions/certgen/scanner?v=4",
			want:   http.StatusMethodNotAllowed,
		},
		"should reject scanner v2": {
			method: http.MethodPost,
			rawURL: "/api/extensions/certgen/scanner",
			want:   http.StatusBadRequest,
		},
		"should reject unknown version": {
			method: http.MethodPost,
			rawURL: "/api/extensions/certgen/scanner?v=2",
			want:   http.StatusBadRequest,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.rawURL, nil)
			rec := httptest.NewRecorder()
			s.scannerHandler(rec, req)
			assert.Equal(t, tc.want, rec.Code)
		})
	}
}
