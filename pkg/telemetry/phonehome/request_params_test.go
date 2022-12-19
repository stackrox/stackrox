package phonehome

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestParams_GetMethod(t *testing.T) {
	methods := map[string]string{
		"Get":     http.MethodGet,
		"Post":    http.MethodPost,
		"Put":     http.MethodPut,
		"Delete":  http.MethodDelete,
		"Patch":   http.MethodPatch,
		"Head":    http.MethodHead,
		"Connect": http.MethodConnect,
		"Options": http.MethodOptions,
		"Trace":   http.MethodTrace,
		"Sixth":   http.MethodGet,
	}
	for prefix, method := range methods {
		rp := &RequestParams{
			Path: "/v1.Service/" + prefix + "Finger",
		}
		assert.Equal(t, method, rp.GetMethod())
	}
	rp := &RequestParams{
		Path: "",
	}
	assert.Equal(t, http.MethodGet, rp.GetMethod())
	rp = &RequestParams{
		Path: "PutFinger",
	}
	assert.Equal(t, http.MethodPut, rp.GetMethod())
}
