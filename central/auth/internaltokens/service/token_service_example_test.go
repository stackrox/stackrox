package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stackrox/rox/central/apitoken/backend"
	apiDS "github.com/stackrox/rox/central/apitoken/datastore"
	apiService "github.com/stackrox/rox/central/apitoken/service"
	"github.com/stackrox/rox/central/role"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	internalV1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	stackroxjwt "github.com/stackrox/rox/pkg/jwt"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const exampleAPITokenSourceID = "https://stackrox.io/jwt-sources#api-tokens"

type exampleAPITokenStore struct {
	apiDS.DataStore
}

// The backend stores issued token metadata here, but the example only needs the JWT.
func (exampleAPITokenStore) AddToken(_ context.Context, _ *storage.TokenMetadata) error {
	return nil
}

type exampleTokenRoleStore struct {
	roleDS.DataStore
	role permissions.ResolvedRole
}

func (s exampleTokenRoleStore) GetAndResolveRole(_ context.Context, name string) (permissions.ResolvedRole, error) {
	if name != s.role.GetRoleName() {
		return nil, fmt.Errorf("unknown role %q", name)
	}
	return s.role, nil
}

type examplePermissionSetStore struct {
	roleDS.DataStore
}

func (examplePermissionSetStore) GetPermissionSetsFiltered(
	_ context.Context, filter func(*storage.PermissionSet) bool,
) ([]*storage.PermissionSet, error) {
	set := &storage.PermissionSet{
		Name: "Image Reader", ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_ACCESS},
	}
	if filter(set) {
		return []*storage.PermissionSet{set}, nil
	}
	return nil, nil
}

type exampleTokenIdentity struct {
	authn.Identity
	roles   []permissions.ResolvedRole
	service *storage.ServiceIdentity
}

func (i exampleTokenIdentity) Roles() []permissions.ResolvedRole { return i.roles }
func (exampleTokenIdentity) UID() string                         { return "example-user" }
func (i exampleTokenIdentity) Service() *storage.ServiceIdentity { return i.service }
func (exampleTokenIdentity) ExternalAuthProvider() authproviders.Provider {
	return nil
}

// A far-future expiry keeps the examples runnable while making exp deterministic.
func exampleExpiry() time.Time {
	return time.Date(2100, time.January, 1, 0, 0, 0, 0, time.UTC)
}

func exampleTokenFactory() tokens.IssuerFactory {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	factory, _, err := tokens.CreateIssuerFactoryAndValidator(
		"https://stackrox.io/jwt", key, "example-key", tokens.WithDefaultTTL(time.Hour))
	if err != nil {
		panic(err)
	}
	return factory
}

func exampleAPIService() (apiService.Service, context.Context) {
	factory := exampleTokenFactory()
	provider := tokenbased.NewTokenAuthProvider(exampleAPITokenSourceID, "api-token", "api-token",
		tokenbased.WithRevocationLayer(tokens.NewRevocationLayer()))
	apiIssuer, err := factory.CreateIssuer(provider)
	if err != nil {
		panic(err)
	}
	analyst := roletest.NewResolvedRole("Analyst", map[string]storage.Access{}, role.AccessScopeIncludeAll)
	api := apiService.New(backend.New(apiIssuer, provider, exampleAPITokenStore{}), exampleTokenRoleStore{role: analyst})
	// ContextWithIdentity uses the testing.TB argument only to mark it as a test helper.
	userCtx := authn.ContextWithIdentity(context.Background(), exampleTokenIdentity{
		roles: []permissions.ResolvedRole{analyst},
	}, nil)
	return api, userCtx
}

func exampleInternalService() (*serviceImpl, context.Context) {
	factory := exampleTokenFactory()
	internalIssuer, err := factory.CreateIssuer(getTokenSource())
	if err != nil {
		panic(err)
	}
	internal := &serviceImpl{
		issuer: internalIssuer, roleManager: &roleManager{}, policy: defaultTokenPolicy(),
		// The internal request specifies a lifetime, so fix the service clock to produce the same expiry.
		now: func() time.Time { return exampleExpiry().Add(-time.Hour) },
	}
	sensorCtx := authn.ContextWithIdentity(context.Background(), exampleTokenIdentity{
		service: &storage.ServiceIdentity{Id: "example-cluster", Type: storage.ServiceType_SENSOR_SERVICE},
	}, nil)
	return internal, sensorCtx
}

// ExampleService_apiToken shows an API token without HTTP restrictions.
func ExampleService_apiToken() {
	api, ctx := exampleAPIService()
	request := &apiV1.GenerateTokenRequest{
		Name: "example API token", Roles: []string{"Analyst"}, Expiration: timestamppb.New(exampleExpiry()),
	}
	response, err := api.GenerateToken(ctx, request)
	if err != nil {
		panic(err)
	}
	printExampleToken(request, response.GetToken())

	// Output:
	// Request:
	// {
	//   "name": "example API token",
	//   "roles": [
	//     "Analyst"
	//   ],
	//   "expiration": {
	//     "seconds": 4102444800
	//   }
	// }
	// JWT payload (excluding iat and jti):
	// {
	//   "aud": "https://stackrox.io/jwt-sources#api-tokens",
	//   "exp": 4102444800,
	//   "iss": "https://stackrox.io/jwt",
	//   "name": "example API token",
	//   "roles": [
	//     "Analyst"
	//   ]
	// }
}

// ExampleService_scopedAPIToken shows an API token with HTTP restrictions.
func ExampleService_scopedAPIToken() {
	api, ctx := exampleAPIService()
	request := &apiV1.GenerateTokenRequest{
		Name: "example scoped API token", Roles: []string{"Analyst"}, Expiration: timestamppb.New(exampleExpiry()),
		HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/images", Methods: []string{"GET"}}},
	}
	response, err := api.GenerateToken(ctx, request)
	if err != nil {
		panic(err)
	}
	printExampleToken(request, response.GetToken())

	// Output:
	// Request:
	// {
	//   "name": "example scoped API token",
	//   "roles": [
	//     "Analyst"
	//   ],
	//   "expiration": {
	//     "seconds": 4102444800
	//   },
	//   "http_request_scopes": [
	//     {
	//       "path": "/v1/images",
	//       "methods": [
	//         "GET"
	//       ]
	//     }
	//   ]
	// }
	// JWT payload (excluding iat and jti):
	// {
	//   "aud": [
	//     "https://stackrox.io/jwt-sources#api-tokens",
	//     "https://stackrox.io/jwt-sources#scoped-rox-tokens"
	//   ],
	//   "exp": 4102444800,
	//   "http": {
	//     "/v1/images": [
	//       "GET"
	//     ]
	//   },
	//   "iss": "https://stackrox.io/jwt",
	//   "name": "example scoped API token",
	//   "roles": [
	//     "Analyst"
	//   ]
	// }
}

// ExampleService_internalToken shows an internal token without HTTP restrictions.
func ExampleService_internalToken() {
	internal, ctx := exampleInternalService()
	request := &internalV1.GenerateTokenForPermissionsAndScopeRequest{
		Permissions:   map[string]internalV1.Access{"Image": internalV1.Access_READ_ACCESS},
		ClusterScopes: []*internalV1.ClusterScope{{ClusterId: "example-cluster", FullClusterAccess: true}},
		Lifetime:      durationpb.New(time.Hour), Requester: "example-sensor",
	}
	response, err := internal.GenerateTokenForPermissionsAndScope(ctx, request)
	if err != nil {
		panic(err)
	}
	printExampleToken(request, response.GetToken())

	// Output:
	// Request:
	// {
	//   "permissions": {
	//     "Image": 1
	//   },
	//   "cluster_scopes": [
	//     {
	//       "cluster_id": "example-cluster",
	//       "full_cluster_access": true
	//     }
	//   ],
	//   "lifetime": {
	//     "seconds": 3600
	//   },
	//   "requester": "example-sensor"
	// }
	// JWT payload (excluding iat and jti):
	// {
	//   "access": [
	//     {
	//       "clusters": {
	//         "example-cluster": [
	//           "*"
	//         ]
	//       },
	//       "name": "internal role",
	//       "permissions": {
	//         "read": [
	//           "Image"
	//         ]
	//       }
	//     }
	//   ],
	//   "aud": "https://stackrox.io/jwt-sources#internal-rox-tokens",
	//   "exp": 4102444800,
	//   "iss": "https://stackrox.io/jwt",
	//   "name": "Generated claims for role internal role expiring at 2100-01-01T00:00:00Z",
	//   "req": "example-sensor"
	// }
}

// ExampleService_scopedInternalToken shows an internal token with HTTP restrictions.
func ExampleService_scopedInternalToken() {
	internal, ctx := exampleInternalService()
	internal.roleManager.roleStore = examplePermissionSetStore{}
	request := &internalV1.GenerateTokenForPermissionsAndScopeRequest{
		ClusterScopes: []*internalV1.ClusterScope{{ClusterId: "example-cluster", FullClusterAccess: true}},
		Lifetime:      durationpb.New(time.Hour), Requester: "example-sensor",
		ScopedPermissionSets: []*internalV1.ScopedPermissionSet{{
			PermissionSetName: "Image Reader",
			HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/images", Methods: []string{"GET"}}},
		}},
	}
	response, err := internal.GenerateTokenForPermissionsAndScope(ctx, request)
	if err != nil {
		panic(err)
	}
	printExampleToken(request, response.GetToken())

	// Output:
	// Request:
	// {
	//   "cluster_scopes": [
	//     {
	//       "cluster_id": "example-cluster",
	//       "full_cluster_access": true
	//     }
	//   ],
	//   "lifetime": {
	//     "seconds": 3600
	//   },
	//   "requester": "example-sensor",
	//   "scoped_permission_sets": [
	//     {
	//       "permission_set_name": "Image Reader",
	//       "http_request_scopes": [
	//         {
	//           "path": "/v1/images",
	//           "methods": [
	//             "GET"
	//           ]
	//         }
	//       ]
	//     }
	//   ]
	// }
	// JWT payload (excluding iat and jti):
	// {
	//   "access": [
	//     {
	//       "http": {
	//         "/v1/images": [
	//           "GET"
	//         ]
	//       },
	//       "name": "internal role",
	//       "permissions": {
	//         "read": [
	//           "Image"
	//         ]
	//       }
	//     }
	//   ],
	//   "aud": [
	//     "https://stackrox.io/jwt-sources#internal-rox-tokens",
	//     "https://stackrox.io/jwt-sources#scoped-rox-tokens"
	//   ],
	//   "clusters": {
	//     "example-cluster": [
	//       "*"
	//     ]
	//   },
	//   "exp": 4102444800,
	//   "iss": "https://stackrox.io/jwt",
	//   "name": "Generated claims for role internal role expiring at 2100-01-01T00:00:00Z",
	//   "req": "example-sensor"
	// }
}

// printExampleToken displays the JWT payload without its variable iat and jti claims.
func printExampleToken(request proto.Message, rawToken string) {
	token, err := stackroxjwt.ParseSigned(rawToken)
	if err != nil {
		panic(err)
	}
	var claims map[string]json.RawMessage
	if err := token.UnsafeClaimsWithoutVerification(&claims); err != nil {
		panic(err)
	}
	for _, field := range []string{"iat", "jti"} {
		delete(claims, field)
	}
	requestJSON, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		panic(err)
	}
	claimsJSON, err := json.MarshalIndent(claims, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Request:\n%s\nJWT payload (excluding iat and jti):\n%s\n", requestJSON, claimsJSON)
}
