package sac

import (
	"context"
	"math/big"
	"testing"
	"time"

	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/sac/authorizer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	authnService "github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	firstClusterID    = "smoke-cluster-1"
	firstClusterName  = "SmokeClusterOne"
	secondClusterID   = "smoke-cluster-2"
	secondClusterName = "SmokeClusterTwo"

	firstNamespaceName  = "SmokeNamespaceOne"
	secondNamespaceName = "SmokeNamespaceTwo"
)

var (
	smokeClusters = []effectiveaccessscope.Cluster{
		&storage.Cluster{Id: firstClusterID, Name: firstClusterName},
		&storage.Cluster{Id: secondClusterID, Name: secondClusterName},
	}

	smokeNamespaces = []effectiveaccessscope.Namespace{
		&storage.NamespaceMetadata{
			Id:          "smoke-namespace-1",
			Name:        firstNamespaceName,
			ClusterId:   firstClusterID,
			ClusterName: firstClusterName,
		},
		&storage.NamespaceMetadata{
			Id:          "smoke-namespace-2",
			Name:        secondNamespaceName,
			ClusterId:   firstClusterID,
			ClusterName: firstClusterName,
		},
	}
)

func TestRootScopeCheckerCoreContextEnricher(t *testing.T) {
	suite.Run(t, new(rootScopeCheckerCoreContextEnricherTestSuite))
}

type rootScopeCheckerCoreContextEnricherTestSuite struct {
	suite.Suite

	authProvider authproviders.Provider
}

func (s *rootScopeCheckerCoreContextEnricherTestSuite) SetupSuite() {
	var err error
	s.authProvider, err = authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Smoke Test Auth Provider"),
	)
	s.Require().NoError(err)
}

func (s *rootScopeCheckerCoreContextEnricherTestSuite) SetupTest() {
	// The role-based scope checker consults the scope inventory, which
	// otherwise requires database access.
	authorizer.SetScopeInventoryForTest(s.T(), smokeClusters, smokeNamespaces)
}

// smokeUserIDentity is a plain user identity: neither a service nor a local
// admin, with the given resolved roles as its only variable part.
type smokeUserIDentity struct {
	roles []permissions.ResolvedRole
}

func (id *smokeUserIDentity) UID() string          { return "smoke-user" }
func (id *smokeUserIDentity) FriendlyName() string { return "Smoke User" }
func (id *smokeUserIDentity) FullName() string     { return "Smoke User" }
func (id *smokeUserIDentity) User() *storage.UserInfo {
	return &storage.UserInfo{Username: "smoke-user"}
}
func (id *smokeUserIDentity) Permissions() map[string]storage.Access { return nil }
func (id *smokeUserIDentity) Roles() []permissions.ResolvedRole      { return id.roles }
func (id *smokeUserIDentity) Service() *storage.ServiceIdentity      { return nil }
func (id *smokeUserIDentity) Attributes() map[string][]string        { return nil }
func (id *smokeUserIDentity) ValidityPeriod() (time.Time, time.Time) {
	return time.Now(), time.Now().Add(time.Hour)
}
func (id *smokeUserIDentity) ExternalAuthProvider() authproviders.Provider { return nil }

func newServiceIdentity() authn.Identity {
	return authnService.WrapMTLSIdentity(mtls.Identity{
		Subject: mtls.NewSubject("smoke-sensor", storage.ServiceType_SENSOR_SERVICE),
		Serial:  big.NewInt(42),
	})
}

func deploymentReadPermissions() map[string]storage.Access {
	return map[string]storage.Access{
		string(resources.Deployment.Resource): storage.Access_READ_ACCESS,
	}
}

func firstNamespaceAccessScope() *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id: "smoke-first-namespace-scope",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{
				ClusterName:   firstClusterName,
				NamespaceName: firstNamespaceName,
			}},
		},
	}
}

// scopeCheck is a single scope checker core probe: whether `access` on
// `resource` is allowed in the scope given by `keys` (no keys means checking
// on the global level).
type scopeCheck struct {
	name     string
	resource permissions.ResourceMetadata
	access   storage.Access
	keys     []sac.ScopeKey
	allowed  bool
}

func newScopeCheck(name string, resource permissions.ResourceMetadata, access storage.Access, allowed bool, keys ...sac.ScopeKey) scopeCheck {
	return scopeCheck{
		name:     name,
		resource: resource,
		access:   access,
		keys:     keys,
		allowed:  allowed,
	}
}

func inNamespace(clusterID, namespaceName string) []sac.ScopeKey {
	return []sac.ScopeKey{sac.ClusterScopeKey(clusterID), sac.NamespaceScopeKey(namespaceName)}
}

func (s *rootScopeCheckerCoreContextEnricherTestSuite) TestGetPreAuthContextEnricher() {
	deploymentViewer := roletest.NewResolvedRole("SmokeDeploymentViewer", deploymentReadPermissions(), firstNamespaceAccessScope())
	unrestrictedViewer := roletest.NewResolvedRole("SmokeUnrestrictedViewer", deploymentReadPermissions(), rolePkg.AccessScopeIncludeAll)

	tests := map[string]struct {
		ctx    context.Context
		checks []scopeCheck
	}{
		"context without identity yields a deny-all scope checker": {
			ctx: context.Background(),
			checks: []scopeCheck{
				newScopeCheck("deny deployment read in namespace", resources.Deployment, storage.Access_READ_ACCESS, false,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("deny deployment write in namespace", resources.Deployment, storage.Access_READ_WRITE_ACCESS, false,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("deny administration read globally", resources.Administration, storage.Access_READ_ACCESS, false),
			},
		},
		"service identity yields an allow-all scope checker": {
			ctx: authn.ContextWithIdentity(context.Background(), newServiceIdentity(), s.T()),
			checks: []scopeCheck{
				newScopeCheck("allow deployment read in namespace", resources.Deployment, storage.Access_READ_ACCESS, true,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("allow deployment write in namespace", resources.Deployment, storage.Access_READ_WRITE_ACCESS, true,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("allow deployment read in unknown cluster", resources.Deployment, storage.Access_READ_ACCESS, true,
					inNamespace("unknown-cluster-id", "UnknownNamespace")...),
				newScopeCheck("allow administration write globally", resources.Administration, storage.Access_READ_WRITE_ACCESS, true),
			},
		},
		"basic auth identity (local admin) yields an allow-all scope checker": {
			ctx: basic.ContextWithAdminIdentity(s.T(), s.authProvider),
			checks: []scopeCheck{
				newScopeCheck("allow deployment read in namespace", resources.Deployment, storage.Access_READ_ACCESS, true,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("allow deployment write in namespace", resources.Deployment, storage.Access_READ_WRITE_ACCESS, true,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("allow administration write globally", resources.Administration, storage.Access_READ_WRITE_ACCESS, true),
			},
		},
		"user identity without roles yields a deny-all scope checker": {
			ctx: authn.ContextWithIdentity(context.Background(), &smokeUserIDentity{}, s.T()),
			checks: []scopeCheck{
				newScopeCheck("deny deployment read in namespace", resources.Deployment, storage.Access_READ_ACCESS, false,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("deny deployment write in namespace", resources.Deployment, storage.Access_READ_WRITE_ACCESS, false,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("deny administration read globally", resources.Administration, storage.Access_READ_ACCESS, false),
			},
		},
		"user identity with namespace-scoped role is restricted to the role": {
			ctx: authn.ContextWithIdentity(context.Background(), &smokeUserIDentity{
				roles: []permissions.ResolvedRole{deploymentViewer},
			}, s.T()),
			checks: []scopeCheck{
				newScopeCheck("allow deployment read in included namespace", resources.Deployment, storage.Access_READ_ACCESS, true,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("deny deployment read in other namespace of same cluster", resources.Deployment, storage.Access_READ_ACCESS, false,
					inNamespace(firstClusterID, secondNamespaceName)...),
				newScopeCheck("deny deployment read in other cluster", resources.Deployment, storage.Access_READ_ACCESS, false,
					inNamespace(secondClusterID, firstNamespaceName)...),
				newScopeCheck("deny deployment write in included namespace", resources.Deployment, storage.Access_READ_WRITE_ACCESS, false,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("deny cluster read without permission", resources.Cluster, storage.Access_READ_ACCESS, false,
					sac.ClusterScopeKey(firstClusterID)),
				newScopeCheck("deny deployment read on global level", resources.Deployment, storage.Access_READ_ACCESS, false),
			},
		},
		"user identity with unrestricted role can read anywhere but not write": {
			ctx: authn.ContextWithIdentity(context.Background(), &smokeUserIDentity{
				roles: []permissions.ResolvedRole{unrestrictedViewer},
			}, s.T()),
			checks: []scopeCheck{
				newScopeCheck("allow deployment read in first namespace", resources.Deployment, storage.Access_READ_ACCESS, true,
					inNamespace(firstClusterID, firstNamespaceName)...),
				newScopeCheck("allow deployment read in other cluster", resources.Deployment, storage.Access_READ_ACCESS, true,
					inNamespace(secondClusterID, firstNamespaceName)...),
				newScopeCheck("allow deployment read on global level", resources.Deployment, storage.Access_READ_ACCESS, true),
				newScopeCheck("deny deployment write in first namespace", resources.Deployment, storage.Access_READ_WRITE_ACCESS, false,
					inNamespace(firstClusterID, firstNamespaceName)...),
			},
		},
	}
	for name, tc := range tests {
		s.Run(name, func() {
			enricher := GetEnricher().GetPreAuthContextEnricher(observe.NewAuthzTraceSink())
			enrichedCtx, err := enricher(tc.ctx)
			s.Require().NoError(err)

			for _, check := range tc.checks {
				s.Run(check.name, func() {
					allowed, err := sac.ForResource(check.resource).AccessAllowed(enrichedCtx, check.access, check.keys...)
					s.NoError(err)
					s.Equal(check.allowed, allowed)
				})
			}
		})
	}
}

func (s *rootScopeCheckerCoreContextEnricherTestSuite) TestAuthzTraceAddedOnlyForActiveSink() {
	enricher := GetEnricher().GetPreAuthContextEnricher(observe.NewAuthzTraceSink())
	ctx := basic.ContextWithAdminIdentity(s.T(), s.authProvider)

	enrichedCtx, err := enricher(ctx)
	s.Require().NoError(err)
	s.Nil(observe.AuthzTraceFromContext(enrichedCtx), "expected no authz trace while the sink has no subscribers")

	sink := observe.NewAuthzTraceSink()
	_ = sink.Subscribe(context.Background())
	enricher = GetEnricher().GetPreAuthContextEnricher(sink)
	enrichedCtx, err = enricher(ctx)
	s.Require().NoError(err)
	s.NotNil(observe.AuthzTraceFromContext(enrichedCtx), "expected an authz trace while the sink is active")
}

func (s *rootScopeCheckerCoreContextEnricherTestSuite) TestGetEnricherReturnsSingleton() {
	s.Same(GetEnricher(), GetEnricher())
}
