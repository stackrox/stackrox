package tokens

import (
	"fmt"
	"net/url"
	"slices"
	"testing"

	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePerAccessHTTPScopes(t *testing.T) {
	valid := func() RoxClaims {
		return RoxClaims{
			InternalRoles: []*InternalRole{
				{HTTPRequestScopes: HTTPRequestScopes{"/v1/": {"GET"}}, Clusters: ClusterScopes{"cluster": {"namespace"}}},
				{HTTPRequestScopes: HTTPRequestScopes{"/v1/images": {"POST"}}, Clusters: ClusterScopes{"other": {"*"}}},
			},
		}
	}
	claims := valid()
	perAccess, err := ValidatePerAccessHTTPScopes(&claims)
	require.NoError(t, err)
	assert.True(t, perAccess)

	for name, change := range map[string]func(*RoxClaims){
		"empty methods deny": func(c *RoxClaims) { c.InternalRoles[1].HTTPRequestScopes["/v1/images"] = []string{} },
		"nil methods deny":   func(c *RoxClaims) { c.InternalRoles[1].HTTPRequestScopes["/v1/images"] = nil },
		"global HTTP scope": func(c *RoxClaims) {
			c.HTTPRequestScopes = HTTPRequestScopes{"/v1/": {"GET"}}
		},
		"same route across roles": func(c *RoxClaims) {
			c.InternalRoles[1].HTTPRequestScopes["/v1/"] = []string{"GET"}
		},
		"unscoped first": func(c *RoxClaims) { c.InternalRoles[0].HTTPRequestScopes = nil },
		"unscoped last":  func(c *RoxClaims) { c.InternalRoles[1].HTTPRequestScopes = nil },
	} {
		t.Run(name, func(t *testing.T) {
			claims := valid()
			change(&claims)
			perAccess, err := ValidatePerAccessHTTPScopes(&claims)
			require.NoError(t, err)
			assert.True(t, perAccess)
		})
	}

	for name, change := range map[string]func(*RoxClaims){
		"empty scope":    func(c *RoxClaims) { c.InternalRoles[1].HTTPRequestScopes = HTTPRequestScopes{} },
		"nil role":       func(c *RoxClaims) { c.InternalRoles[1] = nil },
		"invalid path":   func(c *RoxClaims) { c.InternalRoles[0].HTTPRequestScopes = HTTPRequestScopes{"/%76": {"GET"}} },
		"invalid method": func(c *RoxClaims) { c.InternalRoles[0].HTTPRequestScopes = HTTPRequestScopes{"/": {"Get"}} },
		"duplicate method within role": func(c *RoxClaims) {
			c.InternalRoles[0].HTTPRequestScopes = HTTPRequestScopes{"/": {"GET", "GET"}}
		},
		"too many roles including unscoped": func(c *RoxClaims) {
			for range 31 {
				c.InternalRoles = append(c.InternalRoles, &InternalRole{})
			}
		},
		"too many paths across roles": func(c *RoxClaims) {
			for i := range 31 {
				c.InternalRoles[0].HTTPRequestScopes[fmt.Sprintf("/path%d", i)] = []string{"GET"}
			}
		},
		"too many methods across roles": func(c *RoxClaims) {
			for i := range 16 {
				c.InternalRoles[0].HTTPRequestScopes[fmt.Sprintf("/path%d", i)] = []string{"GET"}
				c.InternalRoles[1].HTTPRequestScopes[fmt.Sprintf("/path%d", i)] = []string{"GET"}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			claims := valid()
			change(&claims)
			_, err := ValidatePerAccessHTTPScopes(&claims)
			require.Error(t, err)
		})
	}
}

func TestScopedRolesForHTTPRequestMixedScopes(t *testing.T) {
	unscoped := &InternalRole{RoleName: "unscoped"}
	broad := &InternalRole{RoleName: "broad", HTTPRequestScopes: HTTPRequestScopes{"/v1/": {"GET"}}}
	specific := &InternalRole{RoleName: "specific", HTTPRequestScopes: HTTPRequestScopes{"/v1/images": {"POST"}}}
	sameRoute := &InternalRole{RoleName: "same route", HTTPRequestScopes: HTTPRequestScopes{"/v1/images": {"POST"}}}
	for order, roles := range map[string][]*InternalRole{
		"unscoped first": {unscoped, broad, specific, sameRoute},
		"unscoped last":  {broad, specific, sameRoute, unscoped},
	} {
		t.Run(order, func(t *testing.T) {
			for name, tc := range map[string]struct {
				request  *requestinfo.HTTPRequest
				selected []*InternalRole
			}{
				"broad": {
					request:  &requestinfo.HTTPRequest{Method: "GET", URL: &url.URL{Path: "/v1/deployments"}},
					selected: []*InternalRole{unscoped, broad},
				},
				"same route union": {
					request:  &requestinfo.HTTPRequest{Method: "POST", URL: &url.URL{Path: "/v1/images"}},
					selected: []*InternalRole{unscoped, specific, sameRoute},
				},
				"no fallback": {
					request:  &requestinfo.HTTPRequest{Method: "GET", URL: &url.URL{Path: "/v1/images"}},
					selected: []*InternalRole{unscoped},
				},
				"unmatched": {
					request:  &requestinfo.HTTPRequest{Method: "GET", URL: &url.URL{Path: "/other"}},
					selected: []*InternalRole{unscoped},
				},
				"encoded path": {
					request: &requestinfo.HTTPRequest{Method: "POST",
						URL: &url.URL{Path: "/v1/images", RawPath: "/v1/%69mages"}},
					selected: []*InternalRole{unscoped},
				},
				"direct gRPC": {selected: []*InternalRole{unscoped}},
			} {
				t.Run(name, func(t *testing.T) {
					before := slices.Clone(roles)
					selected := ScopedRolesForHTTPRequest(roles, requestinfo.RequestInfo{HTTPRequest: tc.request})
					assert.ElementsMatch(t, tc.selected, selected)
					assert.Equal(t, before, roles)
				})
			}
		})
	}
}

func TestScopedRolesForHTTPRequestMostSpecificPath(t *testing.T) {
	roles := []*InternalRole{
		{RoleName: "broad", HTTPRequestScopes: HTTPRequestScopes{"/v1/": {"GET"}}},
		{RoleName: "specific", HTTPRequestScopes: HTTPRequestScopes{"/v1/images": {"POST"}}},
		{RoleName: "denied", HTTPRequestScopes: HTTPRequestScopes{"/v1/blocked/": {}}},
	}
	for name, tc := range map[string]struct {
		method, path, rawPath string
		selected              string
	}{
		"broad":           {method: "GET", path: "/v1/deployments", selected: "broad"},
		"specific":        {method: "POST", path: "/v1/images", selected: "specific"},
		"no fallback":     {method: "GET", path: "/v1/images"},
		"encoded path":    {method: "POST", path: "/v1/images", rawPath: "/v1/%69mages"},
		"unmatched":       {method: "GET", path: "/other"},
		"explicit denial": {method: "GET", path: "/v1/blocked/item"},
	} {
		t.Run(name, func(t *testing.T) {
			selected := ScopedRolesForHTTPRequest(roles, requestinfo.RequestInfo{HTTPRequest: &requestinfo.HTTPRequest{
				Method: tc.method, URL: &url.URL{Path: tc.path, RawPath: tc.rawPath},
			}})
			if tc.selected == "" {
				assert.Empty(t, selected)
			} else {
				require.Len(t, selected, 1)
				assert.Equal(t, tc.selected, selected[0].RoleName)
			}
		})
	}
	assert.Empty(t, ScopedRolesForHTTPRequest(roles, requestinfo.RequestInfo{}))
}
