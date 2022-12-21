package phonehome

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestParams_Is(t *testing.T) {
	assert.True(t, (&RequestParams{}).
		Is(&ServiceMethod{}),
	)
	assert.True(t, (&RequestParams{Method: "/v1.service/method"}).
		Is(&ServiceMethod{GRPCMethod: "/v1.service/method"}),
	)
	assert.True(t, (&RequestParams{Method: "CONNECT", Path: "/v1/method"}).
		Is(&ServiceMethod{HTTPMethod: http.MethodConnect, HTTPPath: "/v1/method"}),
	)
	assert.True(t, (&RequestParams{Method: "CONNECT", Path: "/v1/method"}).
		Is(&ServiceMethod{GRPCMethod: "different", HTTPMethod: http.MethodConnect, HTTPPath: "/v1/method"}),
	)

	assert.False(t, (&RequestParams{Method: "some path"}).
		Is(&ServiceMethod{}),
	)
	assert.False(t, (&RequestParams{Path: "/v2.service/method"}).
		Is(&ServiceMethod{GRPCMethod: "/v1.service/method"}),
	)
	assert.False(t, (&RequestParams{Method: "CONNECT", Path: "/v2/method"}).
		Is(&ServiceMethod{HTTPMethod: http.MethodConnect, HTTPPath: "/v1/method"}),
	)
	assert.False(t, (&RequestParams{Method: "GET", Path: "/v1/method"}).
		Is(&ServiceMethod{HTTPMethod: http.MethodConnect, HTTPPath: "/v1/method"}),
	)
	assert.False(t, (&RequestParams{Method: "GET", Path: "/v1/method"}).
		Is(&ServiceMethod{GRPCMethod: "/v1/method"}),
	)
}
