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
		"/path/":        {"GET", "POST"},
		"/path/subpath": {"GET"},
	}
	for name, tc := range map[string]struct {
		method  string
		path    string
		allowed bool
	}{
		"exact path GET":     {method: "GET", path: "/path/subpath", allowed: true},
		"exact path POST":    {method: "POST", path: "/path/subpath"},
		"prefix descendant":  {method: "POST", path: "/path/another", allowed: true},
		"prefix itself":      {method: "POST", path: "/path/", allowed: true},
		"unrelated endpoint": {method: "POST", path: "/pathology"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.allowed, AllowsHTTPRequest(scopes, requestinfo.RequestInfo{
				HTTPRequest: &requestinfo.HTTPRequest{Method: tc.method, URL: &url.URL{Path: tc.path}},
			}))
		})
	}
}

func TestHTTPRequestScopesFromProto(t *testing.T) {
	for name, requested := range map[string][]*v1.HTTPRequestScope{
		"duplicate": {{Method: "GET", Path: "/metrics"}, {Method: "GET", Path: "/metrics"}},
		"invalid":   {{Method: "GET", Path: "/%6detrics"}},
		"nil entry": {nil},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := HTTPRequestScopesFromProto(requested)
			assert.Error(t, err)
		})
	}
	scopes, err := HTTPRequestScopesFromProto([]*v1.HTTPRequestScope{{Method: "GET", Path: "/v1/metadata"}})
	assert.NoError(t, err)
	assert.Equal(t, []HTTPRequestScope{{Method: "GET", Path: "/v1/metadata"}}, scopes)
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
	assert.True(t, ValidHTTPRequestScope(HTTPRequestScope{Method: "GET", Path: "/v1/telemetry/"}))
}

func TestHTTPRequestScopesByPath(t *testing.T) {
	assert.Equal(t, HTTPRequestScopes{
		"/v1/metadata": {"GET", "HEAD"},
		"/api/graphql": {"POST"},
	}, HTTPRequestScopesByPath([]HTTPRequestScope{
		{Method: "GET", Path: "/v1/metadata"},
		{Method: "HEAD", Path: "/v1/metadata"},
		{Method: "POST", Path: "/api/graphql"},
	}))
	assert.Nil(t, HTTPRequestScopesByPath(nil))
}
