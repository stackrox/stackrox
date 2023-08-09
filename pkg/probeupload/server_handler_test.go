package probeupload

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	validPath   = "b6745d795b8497aaf387843dc8aa07463c944d3ad67288389b754daaebea4b62/collector-4.18.0-305.28.1.el8_4.x86_64.ko.gz"
	invalidPath = "a/b/c/d/xyz.jpeg"
)

// Test_probeServerHandler_Sensor tests the behavior of the handler from `NewConnectionAwareProbeHandler`.
// It focuses on testing the probeServerHandler, but not on the impl. of ProbeSource
func Test_probeServerHandler_Sensor(t *testing.T) {
	tests := map[string]struct {
		source             ProbeSource
		isOnline           bool
		reqMethod          string
		reqURL             string
		expectCode         int
		expectBodyContains string
	}{
		"Method other than GET should return code 405": {
			source:             nil,
			isOnline:           true,
			reqMethod:          "POST",
			reqURL:             "/",
			expectCode:         405,
			expectBodyContains: "invalid method",
		},
		"Invalid prefix should return code 400": {
			source:             nil,
			isOnline:           true,
			reqMethod:          "GET",
			reqURL:             "invalid-prefix",
			expectCode:         400,
			expectBodyContains: "invalid",
		},
		"Valid kernel path for non-existing kernel should return code 404": {
			source:             mockLoadProbe{},
			isOnline:           true,
			reqMethod:          "GET",
			reqURL:             "/" + validPath,
			expectCode:         404,
			expectBodyContains: "not found",
		},
		"Invalid kernel path should return code 404": {
			source:             nil,
			isOnline:           true,
			reqMethod:          "GET",
			reqURL:             "/" + invalidPath,
			expectCode:         404,
			expectBodyContains: "not found",
		},
		"Valid kernel path for existing kernel should return code 200 in online mode": {
			source:             mockLoadProbe{availableProbe: validPath},
			isOnline:           true,
			reqMethod:          "GET",
			reqURL:             "/" + validPath,
			expectCode:         200,
			expectBodyContains: "",
		},
		"Valid kernel path for existing kernel should return code 503 in offline mode": {
			source:             mockLoadProbe{availableProbe: validPath},
			isOnline:           false,
			reqMethod:          "GET",
			reqURL:             "/" + validPath,
			expectCode:         503,
			expectBodyContains: "sensor running in offline mode",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := NewConnectionAwareProbeHandler(func(err error) {
				assert.NoError(t, err)
			}, tt.source)
			h.GoOnline()
			if !tt.isOnline {
				h.GoOffline()
			}

			res := httptest.NewRecorder()
			req, err := http.NewRequest(tt.reqMethod, tt.reqURL, nil)
			assert.NoError(t, err)
			h.ServeHTTP(res, req)

			assert.Equal(t, tt.expectCode, res.Result().StatusCode)

			bodyData, err := io.ReadAll(res.Result().Body)
			assert.NoError(t, err)
			defer func(b io.ReadCloser) {
				assert.NoError(t, b.Close())
			}(res.Result().Body)
			assert.Contains(t, string(bodyData), tt.expectBodyContains)

		})
	}
}

// Test_probeServerHandler_Central tests the behavior of the handler created from `NewProbeServerHandler`.
func Test_probeServerHandler_Central(t *testing.T) {
	tests := map[string]struct {
		source             ProbeSource
		reqURL             string
		expectCode         int
		expectBodyContains string
	}{
		"NewProbeServerHandler should return a handler that is in online mode": {
			source:             mockLoadProbe{availableProbe: validPath},
			reqURL:             "/" + validPath,
			expectCode:         200,
			expectBodyContains: "",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := NewProbeServerHandler(func(err error) {
				assert.NoError(t, err)
			}, tt.source)

			res := httptest.NewRecorder()
			req, err := http.NewRequest("GET", tt.reqURL, nil)
			assert.NoError(t, err)
			h.ServeHTTP(res, req)

			assert.Equal(t, tt.expectCode, res.Result().StatusCode)

			bodyData, err := io.ReadAll(res.Result().Body)
			assert.NoError(t, err)
			defer func(b io.ReadCloser) {
				assert.NoError(t, b.Close())
			}(res.Result().Body)
			assert.Contains(t, string(bodyData), tt.expectBodyContains)

		})
	}
}

var _ ProbeSource = (*mockLoadProbe)(nil)

type mockLoadProbe struct {
	availableProbe string
}

func (m mockLoadProbe) LoadProbe(_ context.Context, fileName string) (io.ReadCloser, int64, error) {
	kernelData := &bytes.Buffer{}
	if m.availableProbe == fileName {
		size, err := kernelData.WriteString("I am the kernel")
		return io.NopCloser(kernelData), int64(size), err // simulate finding kernel
	}
	return nil, 0, nil // simulate not finding the kernel
}

func (m mockLoadProbe) IsAvailable(_ context.Context) (bool, error) {
	return true, nil
}
