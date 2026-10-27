package tokens

import (
	"net/url"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
)

func TestAllowsHTTPRequest(t *testing.T) {
	scopes := HTTPRequestScopes{
		"/metrics":       {"GET"},
		"/v1/telemetry/": {"GET"},
		"/api/graphql":   {"POST"},
	}
	for name, tc := range map[string]struct {
		method  string
		path    string
		rawPath string
		allowed bool
	}{
		"metrics":           {method: "GET", path: "/metrics", allowed: true},
		"query":             {method: "POST", path: "/api/graphql", allowed: true},
		"different method":  {method: "POST", path: "/metrics"},
		"different path":    {method: "GET", path: "/v1/metadata"},
		"prefix descendant": {method: "GET", path: "/v1/telemetry/config", allowed: true},
		"prefix confusion":  {method: "GET", path: "/v1/telemetry-other/config"},
		"exact confusion":   {method: "GET", path: "/metrics/other"},
		"dot segment":       {method: "GET", path: "/v1/telemetry/../metadata"},
		"duplicate slash":   {method: "GET", path: "/v1/telemetry//config"},
		"encoded segment":   {method: "GET", path: "/metrics", rawPath: "/%6detrics"},
		"encoded slash":     {method: "GET", path: "/v1/telemetry/config", rawPath: "/v1%2Ftelemetry/config"},
		"backslash":         {method: "GET", path: "/v1/telemetry/\\config"},
	} {
		t.Run(name, func(t *testing.T) {
			ri := requestinfo.RequestInfo{HTTPRequest: &requestinfo.HTTPRequest{
				Method: tc.method,
				URL:    &url.URL{Path: tc.path, RawPath: tc.rawPath},
			}}
			assert.Equal(t, tc.allowed, AllowsHTTPRequest(scopes, ri))
		})
	}
	assert.False(t, AllowsHTTPRequest(scopes, requestinfo.RequestInfo{}))
	assert.False(t, AllowsHTTPRequest(nil, requestinfo.RequestInfo{HTTPRequest: &requestinfo.HTTPRequest{
		Method: "GET", URL: &url.URL{Path: "/metrics"},
	}}))
}

func TestAllowsHTTPRequestMostSpecificPath(t *testing.T) {
	scopes := HTTPRequestScopes{
		"/":              {"GET"},
		"/path/":         {"GET", "POST"},
		"/path/subpath":  {"GET"},
		"/path/blocked/": {},
	}
	for name, tc := range map[string]struct {
		method  string
		path    string
		rawPath string
		allowed bool
	}{
		"root path":            {method: "GET", path: "/", allowed: true},
		"root prefix":          {method: "GET", path: "/other", allowed: true},
		"root wrong method":    {method: "POST", path: "/other"},
		"root encoded path":    {method: "GET", path: "/metrics", rawPath: "/%6detrics"},
		"exact path GET":       {method: "GET", path: "/path/subpath", allowed: true},
		"exact path POST":      {method: "POST", path: "/path/subpath"},
		"prefix descendant":    {method: "POST", path: "/path/another", allowed: true},
		"prefix itself":        {method: "POST", path: "/path/", allowed: true},
		"empty methods GET":    {method: "GET", path: "/path/blocked/item"},
		"empty methods POST":   {method: "POST", path: "/path/blocked/item"},
		"empty methods prefix": {method: "GET", path: "/path/blocked/"},
		"unrelated endpoint":   {method: "POST", path: "/pathology"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.allowed, AllowsHTTPRequest(scopes, requestinfo.RequestInfo{
				HTTPRequest: &requestinfo.HTTPRequest{Method: tc.method, URL: &url.URL{Path: tc.path, RawPath: tc.rawPath}},
			}))
		})
	}
}

func TestHTTPRequestScopesFromProto(t *testing.T) {
	for name, requested := range map[string][]*v1.HTTPRequestScope{
		"duplicate path":   {{Path: "/metrics", Methods: []string{"GET"}}, {Path: "/metrics", Methods: []string{"POST"}}},
		"duplicate method": {{Path: "/metrics", Methods: []string{"GET", "GET"}}},
		"invalid path":     {{Path: "/%6detrics", Methods: []string{"GET"}}},
		"invalid method":   {{Path: "/metrics", Methods: []string{"Get"}}},
		"empty method":     {{Path: "/metrics", Methods: []string{""}}},
		"nil entry":        {nil},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := HTTPRequestScopesFromProto(requested)
			assert.Error(t, err)
		})
	}
	scopes, err := HTTPRequestScopesFromProto([]*v1.HTTPRequestScope{
		{Path: "/v1/metadata", Methods: []string{"GET", "HEAD"}},
		{Path: "/v1/blocked/"},
	})
	assert.NoError(t, err)
	assert.Equal(t, HTTPRequestScopes{
		"/v1/metadata": {"GET", "HEAD"},
		"/v1/blocked/": {},
	}, scopes)
	root, err := HTTPRequestScopesFromProto([]*v1.HTTPRequestScope{{Path: "/", Methods: []string{"GET"}}})
	assert.NoError(t, err)
	assert.Equal(t, HTTPRequestScopes{"/": {"GET"}}, root)
	assert.NotNil(t, scopes["/v1/blocked/"])
	assert.False(t, AllowsHTTPRequest(scopes, requestinfo.RequestInfo{
		HTTPRequest: &requestinfo.HTTPRequest{Method: "GET", URL: &url.URL{Path: "/v1/blocked/item"}},
	}))
	scopes, err = HTTPRequestScopesFromProto(nil)
	assert.NoError(t, err)
	assert.Nil(t, scopes)
}

func TestValidHTTPRequestScope(t *testing.T) {
	for name, scope := range map[string]HTTPRequestScope{
		"empty method":     {Path: "/metrics"},
		"lowercase method": {Method: "Get", Path: "/metrics"},
		"empty path":       {Method: "GET"},
		"relative path":    {Method: "GET", Path: "metrics"},
		"dot segment":      {Method: "GET", Path: "/v1/../metrics"},
		"encoded path":     {Method: "GET", Path: "/%6detrics"},
	} {
		t.Run(name, func(t *testing.T) { assert.False(t, ValidHTTPRequestScope(scope)) })
	}
	assert.True(t, ValidHTTPRequestScope(HTTPRequestScope{Method: "GET", Path: "/metrics"}))
	assert.True(t, ValidHTTPRequestScope(HTTPRequestScope{Method: "GET", Path: "/"}))
	assert.True(t, ValidHTTPRequestScope(HTTPRequestScope{Method: "GET", Path: "/v1/telemetry/"}))
}
