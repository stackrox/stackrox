package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	roleDS "github.com/stackrox/rox/central/role/datastore"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/durationpb"
)

func scopedGrant(name, path string) *v1.SelectedPermissionSet {
	grant := &v1.SelectedPermissionSet{Selector: &v1.SelectedPermissionSet_Name{Name: name}}
	if path != "" {
		grant.HttpRequestScopes = []*apiV1.HTTPRequestScope{{Path: path, Methods: []string{"GET"}}}
	}
	return grant
}

func perAccessRequest() *v1.GenerateTokenForPermissionsAndScopeRequest {
	return &v1.GenerateTokenForPermissionsAndScopeRequest{
		PermissionSets: []*v1.SelectedPermissionSet{scopedGrant("Deployment Reader", "/v1/deployments")},
		ClusterScopes:  []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"prod"}}},
		Lifetime:       durationpb.New(time.Hour), Requester: "per-access-test",
	}
}

func TestPerAccessRequestValidation(t *testing.T) {
	for name, tc := range map[string]struct {
		change func(*v1.GenerateTokenForPermissionsAndScopeRequest)
		valid  bool
	}{
		"scoped":   {valid: true},
		"unscoped": {valid: true, change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) { r.PermissionSets[0].HttpRequestScopes = nil }},
		"denial only": {valid: true, change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets[0].HttpRequestScopes[0].Methods = nil
		}},
		"legacy permissions mixed": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.Permissions = map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS}
		}},
		"nil grant": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets = []*v1.SelectedPermissionSet{nil}
		}},
		"empty selector": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets[0].Selector = &v1.SelectedPermissionSet_Name{}
		}},
		"invalid UUID": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets[0].Selector = &v1.SelectedPermissionSet_Id{Id: "bad"}
		}},
		"foreign cluster":  {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) { r.ClusterScopes[0].ClusterId = "other" }},
		"missing lifetime": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) { r.Lifetime = nil }},
		"nil HTTP scope": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets[0].HttpRequestScopes = []*apiV1.HTTPRequestScope{nil}
		}},
		"encoded path": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets[0].HttpRequestScopes[0].Path = "/v1/%69mages"
		}},
		"invalid method": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets[0].HttpRequestScopes[0].Methods = []string{"get"}
		}},
		"duplicate path": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets[0].HttpRequestScopes = append(r.PermissionSets[0].HttpRequestScopes, r.GetPermissionSets()[0].GetHttpRequestScopes()[0].CloneVT())
		}},
		"33 unscoped grants": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets = nil
			for i := range 33 {
				r.PermissionSets = append(r.PermissionSets, scopedGrant(fmt.Sprint(i), ""))
			}
		}},
		"aggregate HTTP bounds": {change: func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.PermissionSets = []*v1.SelectedPermissionSet{scopedGrant("first", ""), scopedGrant("second", "")}
			for _, grant := range r.GetPermissionSets() {
				for i := range 17 {
					grant.HttpRequestScopes = append(grant.HttpRequestScopes, &apiV1.HTTPRequestScope{Path: fmt.Sprintf("/%d", i), Methods: []string{"GET"}})
				}
			}
		}},
	} {
		t.Run(name, func(t *testing.T) {
			req := perAccessRequest()
			if tc.change != nil {
				tc.change(req)
			}
			_, err := defaultTokenPolicy().enforce(sensorContext(t, gomock.NewController(t), "sensor-cluster"), req)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, errox.InvalidArgs)
			}
		})
	}
}

type permissionSetFixtureStore struct {
	roleDS.DataStore
	sets []*storage.PermissionSet
}

func (s permissionSetFixtureStore) GetPermissionSet(_ context.Context, id string) (*storage.PermissionSet, bool, error) {
	for _, ps := range s.sets {
		if ps.GetId() == id {
			return ps, true, nil
		}
	}
	return nil, false, nil
}

func (s permissionSetFixtureStore) GetPermissionSetsFiltered(_ context.Context, filter func(*storage.PermissionSet) bool) ([]*storage.PermissionSet, error) {
	var matches []*storage.PermissionSet
	for _, ps := range s.sets {
		if filter(ps) {
			matches = append(matches, ps)
		}
	}
	return matches, nil
}

func perAccessStore() permissionSetFixtureStore {
	first, second := constrainedPermissionSet(), constrainedPermissionSet()
	first.Name = "Deployment Reader"
	first.ResourceToAccess = map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS}
	second.Id = "11111111-2222-4333-8444-555555555555"
	second.Name = "Image Reader"
	second.ResourceToAccess = map[string]storage.Access{"Image": storage.Access_READ_ACCESS}
	return permissionSetFixtureStore{sets: []*storage.PermissionSet{first, second}}
}

func TestPerAccessResolvedAliasesAndUnsafeSets(t *testing.T) {
	for name, tc := range map[string]struct{ alias, duplicate, missing, unsafe bool }{
		"ID and name alias":             {alias: true},
		"duplicate name":                {duplicate: true},
		"missing set":                   {missing: true},
		"unsafe set among valid grants": {unsafe: true},
	} {
		t.Run(name, func(t *testing.T) {
			store := perAccessStore()
			req := perAccessRequest()
			if tc.duplicate {
				req.PermissionSets = append(req.PermissionSets, scopedGrant("Deployment Reader", "/v1/deployments"))
			} else if tc.alias {
				req.PermissionSets = append(req.PermissionSets, &v1.SelectedPermissionSet{Selector: &v1.SelectedPermissionSet_Id{Id: store.sets[0].GetId()}})
			} else {
				req.PermissionSets = append(req.PermissionSets, scopedGrant("Image Reader", "/v1/images"))
			}
			if tc.missing {
				store.sets = store.sets[:1]
			}
			if tc.unsafe {
				store.sets[1].ResourceToAccess["Image"] = storage.Access_READ_WRITE_ACCESS
			}
			svc := &serviceImpl{roleManager: &roleManager{roleStore: store}, policy: defaultTokenPolicy(), now: time.Now}
			response, err := svc.GenerateTokenForPermissionsAndScope(sensorContext(t, gomock.NewController(t), "sensor-cluster"), req)
			assert.Nil(t, response)
			if tc.missing {
				require.ErrorIs(t, err, errox.NotFound)
			} else {
				require.ErrorIs(t, err, errox.InvalidArgs)
			}
		})
	}
}

func TestPerAccessTokenSourcesAndRoleIsolation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	factory, validator, err := tokens.CreateIssuerFactoryAndValidator("test-issuer", key, "test-key")
	require.NoError(t, err)
	issuer, scopedIssuer := getTokenIssuers(factory)
	oldFactory, oldValidator, err := tokens.CreateIssuerFactoryAndValidator("test-issuer", key, "test-key")
	require.NoError(t, err)
	getTokenIssuer(oldFactory)
	issued := make(map[string]*tokens.TokenInfo)
	for name, paths := range map[string][]string{
		"scoped":                   {"/v1/", "/v1/images"},
		"unscoped":                 {"", ""},
		"mixed first unrestricted": {"", "/v1/images"},
		"mixed last unrestricted":  {"/v1/", ""},
	} {
		t.Run(name, func(t *testing.T) {
			req := perAccessRequest()
			req.Lifetime = durationpb.New(2 * time.Hour)
			req.PermissionSets = []*v1.SelectedPermissionSet{scopedGrant("Deployment Reader", paths[0]), scopedGrant("Image Reader", paths[1])}
			if paths[0] != "" {
				req.PermissionSets[0].HttpRequestScopes = append(req.PermissionSets[0].HttpRequestScopes, &apiV1.HTTPRequestScope{Path: "/v1/private/"})
			}
			now := time.Now().Truncate(time.Second)
			svc := &serviceImpl{issuer: issuer, scopedIssuer: scopedIssuer, roleManager: &roleManager{roleStore: perAccessStore()}, policy: defaultTokenPolicy(), now: func() time.Time { return now }}
			response, err := svc.GenerateTokenForPermissionsAndScope(sensorContext(t, gomock.NewController(t), "sensor-cluster"), req)
			require.NoError(t, err)
			info, err := validator.Validate(t.Context(), response.GetToken())
			require.NoError(t, err)
			sourceID := internalTokenId
			if paths[0] != "" || paths[1] != "" {
				sourceID = internalHTTPScopedTokenID
			}
			assert.Equal(t, []string{sourceID}, []string(info.Audience))
			assert.Equal(t, now.Add(time.Hour), info.Expiry())
			assert.Equal(t, req.GetRequester(), info.Requester)
			require.Len(t, info.InternalRoles, 2)
			assert.Equal(t, []string{"Deployment"}, info.InternalRoles[0].Permissions[storage.Access_READ_ACCESS])
			assert.Equal(t, []string{"Image"}, info.InternalRoles[1].Permissions[storage.Access_READ_ACCESS])
			for _, role := range info.InternalRoles {
				assert.Equal(t, tokens.ClusterScopes{"sensor-cluster": {"prod"}}, role.Clusters)
			}
			selected := tokens.ScopedRolesForHTTPRequest(info.InternalRoles, requestinfo.RequestInfo{
				HTTPRequest: &requestinfo.HTTPRequest{Method: http.MethodGet, URL: &url.URL{Path: "/v1/images"}},
			})
			if name == "scoped" {
				assert.Equal(t, info.InternalRoles[1:], selected, "a broader grant must not leak into the more specific route")
				assert.Empty(t, tokens.ScopedRolesForHTTPRequest(info.InternalRoles, requestinfo.RequestInfo{}), "direct gRPC must not use scoped roles")
				assert.Empty(t, tokens.ScopedRolesForHTTPRequest(info.InternalRoles, requestinfo.RequestInfo{
					HTTPRequest: &requestinfo.HTTPRequest{Method: http.MethodGet, URL: &url.URL{Path: "/v1/private/image"}},
				}))
			}
			_, err = oldValidator.Validate(t.Context(), info.Token)
			if sourceID == internalHTTPScopedTokenID {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			otherIssuer := scopedIssuer
			if sourceID == internalHTTPScopedTokenID {
				otherIssuer = issuer
			}
			_, err = otherIssuer.Issue(t.Context(), info.RoxClaims, tokens.WithExpiry(info.Expiry()))
			require.Error(t, err)
			issued[sourceID] = info
		})
	}
	require.Len(t, issued, 2)
	for sourceID, token := range issued {
		otherID := internalTokenId
		if sourceID == internalTokenId {
			otherID = internalHTTPScopedTokenID
		}
		source := issued[otherID].Sources[0].(tokenbased.TokenAuthProvider)
		source.Revoke(token.ID, token.Expiry())
		_, err := validator.Validate(t.Context(), token.Token)
		require.ErrorContains(t, err, "token has been revoked")
	}
}
