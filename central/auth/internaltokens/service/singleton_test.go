package service

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/url"
	"testing"
	"time"

	roleDataStoreMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestGenerateTokenSources(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	factory, validator, err := tokens.CreateIssuerFactoryAndValidator("test-issuer", key, "test-key")
	require.NoError(t, err)
	issuer, scopedIssuer := getTokenIssuers(factory)
	issuedTokens := make(map[string]*tokens.TokenInfo)

	for name, tc := range map[string]struct {
		paths      []string
		deniedPath string
		sourceID   string
	}{
		"legacy permissions": {sourceID: internalTokenId},
		"unscoped roles":     {paths: []string{"", ""}, sourceID: internalTokenId},
		"scoped roles":       {paths: []string{"/v1/deployments", "/v1/images"}, sourceID: internalHTTPScopedTokenID},
		"first unscoped":     {paths: []string{"", "/v1/images"}, sourceID: internalHTTPScopedTokenID},
		"last unscoped":      {paths: []string{"/v1/deployments", ""}, sourceID: internalHTTPScopedTokenID},
		"specific denial overrides broad allow": {
			paths: []string{"/v1/", "/metrics"}, deniedPath: "/v1/images/",
			sourceID: internalHTTPScopedTokenID,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := roleDataStoreMocks.NewMockDataStore(ctrl)
			request := &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1, Namespaces: []string{"namespace A"}}},
				Lifetime:      durationpb.New(time.Minute),
			}
			if len(tc.paths) > 0 {
				names := []string{"Deployment Reader", "Image Reader"}
				expectPermissionSetLookup(t, store, names, []*storage.PermissionSet{
					{Name: names[0], ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS}},
					{Name: names[1], ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_ACCESS}},
				}, nil)
				for i, path := range tc.paths {
					grant := &v1.ScopedPermissionSet{Selector: &v1.ScopedPermissionSet_Name{Name: names[i]}}
					if path != "" {
						grant.HttpRequestScopes = []*apiV1.HTTPRequestScope{{Path: path, Methods: []string{"GET"}}}
					}
					if i == 0 && tc.deniedPath != "" {
						grant.HttpRequestScopes = append(grant.HttpRequestScopes, &apiV1.HTTPRequestScope{Path: tc.deniedPath})
					}
					request.PermissionSets = append(request.PermissionSets, grant)
				}
			}
			svc := &serviceImpl{
				issuer:       issuer,
				scopedIssuer: scopedIssuer,
				roleManager:  &roleManager{roleStore: store},
				now:          time.Now,
				policy:       defaultTokenPolicy(),
			}
			ctx := sensorContext(t, ctrl, fixtureconsts.Cluster1)
			response, err := svc.GenerateTokenForPermissionsAndScope(ctx, request)
			require.NoError(t, err)
			info, err := validator.Validate(ctx, response.GetToken())
			require.NoError(t, err)
			assert.Equal(t, []string{tc.sourceID}, []string(info.Claims.Audience))
			require.Len(t, info.Sources, 1)
			assert.Equal(t, tc.sourceID, info.Sources[0].ID())
			require.Len(t, info.Claims.InternalRoles, max(1, len(tc.paths)))
			clusters := tokens.ClusterScopes{fixtureconsts.Cluster1: {"namespace A"}}
			for i, role := range info.Claims.InternalRoles {
				if len(tc.paths) > 0 && tc.paths[i] != "" {
					expected := tokens.HTTPRequestScopes{tc.paths[i]: {"GET"}}
					if i == 0 && tc.deniedPath != "" {
						expected[tc.deniedPath] = []string{}
					}
					assert.Equal(t, expected, role.HTTPRequestScopes)
				} else {
					assert.Nil(t, role.HTTPRequestScopes)
				}
				assert.Equal(t, clusters, role.Clusters)
			}
			if tc.deniedPath != "" {
				for path, allowed := range map[string]bool{
					"/v1/deployments":       true,
					tc.deniedPath:           false,
					tc.deniedPath + "image": false,
				} {
					selected := tokens.ScopedRolesForHTTPRequest(info.Claims.InternalRoles, requestinfo.RequestInfo{
						HTTPRequest: &requestinfo.HTTPRequest{Method: http.MethodGet, URL: &url.URL{Path: path}},
					})
					if allowed {
						assert.Equal(t, info.Claims.InternalRoles[:1], selected, path)
					} else {
						assert.Empty(t, selected, path)
					}
				}
			}
			otherIssuer := scopedIssuer
			if tc.sourceID == internalHTTPScopedTokenID {
				otherIssuer = issuer
			}
			_, err = otherIssuer.Issue(ctx, info.Claims.RoxClaims, tokens.WithExpiry(info.Expiry()))
			assert.ErrorContains(t, err, "token HTTP scopes do not match source")
			issuedTokens[tc.sourceID] = info
		})
	}

	t.Run("shared revocation", func(t *testing.T) {
		require.Len(t, issuedTokens, 2)
		for sourceID, info := range issuedTokens {
			otherID := internalTokenId
			if sourceID == internalTokenId {
				otherID = internalHTTPScopedTokenID
			}
			otherSource, ok := issuedTokens[otherID].Sources[0].(tokenbased.TokenAuthProvider)
			require.True(t, ok)
			otherSource.Revoke(info.Claims.ID, info.Expiry())
			_, err := validator.Validate(t.Context(), info.Token)
			assert.ErrorContains(t, err, "token has been revoked")
		}
	})
}
