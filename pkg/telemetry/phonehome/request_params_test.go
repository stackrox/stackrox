package phonehome

import (
	"net/http"
	"strings"
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
	assert.True(t, (&RequestParams{Method: "PUT", Path: "/v1/object/id"}).
		Is(&ServiceMethod{HTTPMethod: http.MethodPut, HTTPPath: "/v1/object/*"}),
	)
	assert.True(t, (&RequestParams{Method: "PUT", Path: "/v1/object/id"}).
		Is(&ServiceMethod{GRPCMethod: "different", HTTPMethod: http.MethodPut, HTTPPath: "/v1/object/*"}),
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
	assert.False(t, (&RequestParams{Method: "PUT", Path: "/v1/objects"}).
		Is(&ServiceMethod{HTTPMethod: http.MethodPut, HTTPPath: "/v1/object/id"}),
	)
	assert.False(t, (&RequestParams{Method: "PUT", Path: "/v1/objects"}).
		Is(&ServiceMethod{GRPCMethod: "/v1/objects"}),
	)
	assert.False(t, (&RequestParams{Method: "PUT", Path: "/v1/objects"}).
		Is(&ServiceMethod{GRPCMethod: "/v1/object/*"}),
	)
}

func Test_hasPathIn(t *testing.T) {
	trueCases := []struct {
		path  string
		paths []string
	}{
		{"abc", []string{"abc"}},
		{"abc", []string{"*"}},
		{"abc", []string{"def", "abc"}},
		{"abc", []string{"ab*"}},
		{"abc", []string{"ab*"}},
	}

	rp := RequestParams{}
	for _, pp := range trueCases {
		rp.Path = pp.path
		assert.True(t, rp.HasPathIn(pp.paths), pp.path, " in ", pp.paths)
	}

	falseCases := []struct {
		path  string
		paths []string
	}{
		{"abc", []string{"abcd"}},
		{"abc", []string{"x*"}},
		{"abc", []string{"def", "abcd"}},
		{"abc", []string{"ab*c"}},
		{"abc", []string{"ab"}},
		{"*", []string{"abc"}},
	}

	for _, pp := range falseCases {
		rp.Path = pp.path
		assert.False(t, rp.HasPathIn(pp.paths), pp.path, " in ", pp.paths)
	}
}

func TestHasUserAgentIn(t *testing.T) {
	rp := RequestParams{
		UserAgent: "Some Agent Value",
	}
	tests := map[string]bool{
		"Ogent,Agent,Ugent":  true,
		"Ogent,Xgent,Ugent":  false,
		"Ogent,Agen,Ugent":   true,
		"Ogent,AgentX,Ugent": false,
		"Agen":               true,
		"gent":               true,
		"Some":               true,
		"Value":              true,
		"Some Agent":         true,
		"A,Ag,Age":           true,
	}
	for substrings, match := range tests {
		t.Run(substrings, func(t *testing.T) {
			assert.Equal(t, match, rp.HasUserAgentWith(strings.Split(substrings, ",")))
		})
	}
}
