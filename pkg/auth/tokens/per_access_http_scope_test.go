package tokens

import (
	"net/url"
	"testing"

	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePerAccessHTTPScopes(t *testing.T) {
	valid := func() RoxClaims {
		return RoxClaims{
			Clusters: ClusterScopes{"cluster": {"namespace"}},
			InternalRoles: []*InternalRole{
				{HTTPRequestScopes: HTTPRequestScopes{"/v1/": {"GET"}}},
				{HTTPRequestScopes: HTTPRequestScopes{"/v1/images": {"POST"}}},
			},
		}
	}
	claims := valid()
	perAccess, err := ValidatePerAccessHTTPScopes(&claims)
	require.NoError(t, err)
	assert.True(t, perAccess)

	for name, change := range map[string]func(*RoxClaims){
		"duplicate method and path": func(c *RoxClaims) { c.InternalRoles[1].HTTPRequestScopes["/v1/"] = []string{"GET"} },
		"empty methods":             func(c *RoxClaims) { c.InternalRoles[1].HTTPRequestScopes["/v1/images"] = nil },
		"missing role scope":        func(c *RoxClaims) { c.InternalRoles[1].HTTPRequestScopes = nil },
		"role cluster scope":        func(c *RoxClaims) { c.InternalRoles[0].Clusters = ClusterScopes{"cluster": {"other"}} },
		"global HTTP scope":         func(c *RoxClaims) { c.HTTPRequestScopes = HTTPRequestScopes{"/": {"GET"}} },
		"invalid path":              func(c *RoxClaims) { c.InternalRoles[0].HTTPRequestScopes = HTTPRequestScopes{"/%76": {"GET"}} },
	} {
		t.Run(name, func(t *testing.T) {
			claims := valid()
			change(&claims)
			_, err := ValidatePerAccessHTTPScopes(&claims)
			require.Error(t, err)
		})
	}
}

func TestScopedRolesForHTTPRequestMostSpecificPath(t *testing.T) {
	roles := []*InternalRole{
		{RoleName: "broad", HTTPRequestScopes: HTTPRequestScopes{"/v1/": {"GET"}}},
		{RoleName: "specific", HTTPRequestScopes: HTTPRequestScopes{"/v1/images": {"POST"}}},
	}
	for name, tc := range map[string]struct {
		method, path, rawPath string
		selected              string
	}{
		"broad":        {method: "GET", path: "/v1/deployments", selected: "broad"},
		"specific":     {method: "POST", path: "/v1/images", selected: "specific"},
		"no fallback":  {method: "GET", path: "/v1/images"},
		"encoded path": {method: "POST", path: "/v1/images", rawPath: "/v1/%69mages"},
		"unmatched":    {method: "GET", path: "/other"},
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
