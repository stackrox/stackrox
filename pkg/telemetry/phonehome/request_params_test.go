package phonehome

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestParams_GetMethod(t *testing.T) {
	rp := &RequestParams{Path: "/v1.Service/Method"}
	assert.Equal(t, rp.Path, rp.GetMethod(), "must be equal to Path, as there's no request details")

	rp = &RequestParams{Path: "/v1/method"}
	rp.HTTPReq, _ = http.NewRequest(http.MethodPost, "/path", nil)
	assert.Equal(t, http.MethodPost, rp.GetMethod(), "must be POST, as the HTTP request is provided")

	rp.HTTPReq, _ = http.NewRequest("", "/path", nil)
	assert.Equal(t, http.MethodGet, rp.GetMethod(), "must be GET, as this is the default HTTP method")
}
