package tests

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	resA       = permissions.ResourceMetadata{Resource: "res-A", Scope: permissions.GlobalScope}
	resB       = permissions.ResourceMetadata{Resource: "res-B", Scope: permissions.GlobalScope}
	resC       = permissions.ResourceMetadata{Resource: "res-C", Scope: permissions.GlobalScope}
	resD       = permissions.ResourceMetadata{Resource: "res-D", Scope: permissions.GlobalScope}
	scopedRes  = permissions.ResourceMetadata{Resource: "scoped-res", Scope: permissions.ClusterScope}
	scopedResB = permissions.ResourceMetadata{Resource: "scoped-resB", Scope: permissions.ClusterScope}

	readOnAllRes = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resA, resB)))
	writeOnAllRes = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resA, resB)))
	readWriteOnAllRes = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resA, resB)))
	readOnOneRes = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resA)))
	writeOnOneRes = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resB)))
	readWriteOnOneRes = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resA)))
	noAccessAllRes = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_NO_ACCESS), sac.ResourceScopeKeys(resA, resB)))
	noAccessOneRes = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_NO_ACCESS), sac.ResourceScopeKeys(resA)))
	accessOnOtherRes = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resC, resD)))

	readOnAScopedResWithCorrectScope = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(scopedRes, resD), sac.ClusterScopeKeys("cluster-1")))

	readOnAllScopedResWithCorrectScope = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(scopedRes, scopedResB), sac.ClusterScopeKeys("cluster-1")))
)

type forResourcesHelpersTestSuite struct {
	suite.Suite
}

func TestForResourcesHelpers(t *testing.T) {
	suite.Run(t, new(forResourcesHelpersTestSuite))
}

func (s *forResourcesHelpersTestSuite) TestForAccessToAny() {
	cases := []struct {
		title        string
		ctx          context.Context
		resources    []permissions.ResourceMetadata
		readAllowed  bool
		writeAllowed bool
	}{
		{
			title:        "Can only read when user has read on all the required resources",
			ctx:          readOnAllRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  true,
			writeAllowed: false,
		},
		{
			title:        "Can only write when user has write on all the required resources",
			ctx:          writeOnAllRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: true,
		},
		{
			title:        "Can read and write write when user has read and write on all the required resources",
			ctx:          readWriteOnAllRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  true,
			writeAllowed: true,
		},
		{
			title:        "Can only read when user has read on just one of the required resources",
			ctx:          readOnOneRes,
			resources:    []permissions.ResourceMetadata{resB, resA, resC},
			readAllowed:  true,
			writeAllowed: false,
		},
		{
			title:        "Can only write when user has write on just one of the required resources",
			ctx:          writeOnOneRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: true,
		},
		{
			title:        "Can read and write write when user has read and write on just one of the required resources",
			ctx:          readWriteOnOneRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  true,
			writeAllowed: true,
		},
		{
			title:        "Can't read or write when user has no access on just one of the required resources",
			ctx:          noAccessOneRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has no access on all of the required resources",
			ctx:          noAccessAllRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has access on other resources but not ones specified",
			ctx:          accessOnOtherRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: false,
		},
	}

	for _, c := range cases {
		s.T().Run(c.title, func(t *testing.T) {
			forResourceHelpers := make([]sac.ForResourceHelper, 0, len(c.resources))
			for _, r := range c.resources {
				forResourceHelpers = append(forResourceHelpers, sac.ForResource(r))
			}
			forResources := sac.ForResources(forResourceHelpers...)

			read, err := forResources.ReadAllowedToAny(c.ctx)
			s.NoError(err)
			s.Equal(c.readAllowed, read)

			write, err := forResources.WriteAllowedToAny(c.ctx)
			s.NoError(err)
			s.Equal(c.writeAllowed, write)
		})
	}
}

func (s *forResourcesHelpersTestSuite) TestForAccessToAnyWithScopeKeys() {
	cases := []struct {
		title        string
		ctx          context.Context
		resources    []permissions.ResourceMetadata
		scopeKeys    []sac.ScopeKey
		readAllowed  bool
		writeAllowed bool
	}{
		{
			title:        "Can only read when user has read on all the required resources with global scope",
			ctx:          readOnOneRes,
			resources:    []permissions.ResourceMetadata{resA, resB},
			scopeKeys:    sac.GlobalScopeKey(),
			readAllowed:  true,
			writeAllowed: false,
		},
		{
			title:        "Can only read when user has read on all the required resources with correct scope",
			ctx:          readOnAScopedResWithCorrectScope,
			resources:    []permissions.ResourceMetadata{scopedRes},
			scopeKeys:    []sac.ScopeKey{sac.ClusterScopeKey("cluster-1")},
			readAllowed:  true,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has read on all the required resources with incorrect scope",
			ctx:          readOnAScopedResWithCorrectScope,
			resources:    []permissions.ResourceMetadata{scopedRes, resD},
			scopeKeys:    []sac.ScopeKey{sac.ClusterScopeKey("cluster-2")},
			readAllowed:  false,
			writeAllowed: false,
		},
	}

	for _, c := range cases {
		s.T().Run(c.title, func(t *testing.T) {
			forResourceHelpers := make([]sac.ForResourceHelper, 0, len(c.resources))
			for _, r := range c.resources {
				forResourceHelpers = append(forResourceHelpers, sac.ForResource(r))
			}
			forResources := sac.ForResources(forResourceHelpers...)

			read, err := forResources.ReadAllowedToAny(c.ctx, c.scopeKeys...)
			s.NoError(err)
			s.Equal(c.readAllowed, read)

			write, err := forResources.WriteAllowedToAny(c.ctx, c.scopeKeys...)
			s.NoError(err)
			s.Equal(c.writeAllowed, write)
		})
	}
}

func (s *forResourcesHelpersTestSuite) TestForAccessToAll() {
	cases := []struct {
		title        string
		ctx          context.Context
		resources    []permissions.ResourceMetadata
		readAllowed  bool
		writeAllowed bool
	}{
		{
			title:        "Can only read when user has read on all the required resources",
			ctx:          readOnAllRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  true,
			writeAllowed: false,
		},
		{
			title:        "Can only write when user has write on all the required resources",
			ctx:          writeOnAllRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: true,
		},
		{
			title:        "Can read and write write when user has read and write on all the required resources",
			ctx:          readWriteOnAllRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  true,
			writeAllowed: true,
		},
		{
			title:        "Can't read or write when user has read on just one of the required resources",
			ctx:          readOnOneRes,
			resources:    []permissions.ResourceMetadata{resB, resA, resC},
			readAllowed:  false,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has write on just one of the required resources",
			ctx:          writeOnOneRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has read and write on just one of the required resources",
			ctx:          readWriteOnOneRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has no access on just one of the required resources",
			ctx:          noAccessOneRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has no access on all of the required resources",
			ctx:          noAccessAllRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has access on other resources but not ones specified",
			ctx:          accessOnOtherRes,
			resources:    []permissions.ResourceMetadata{resB, resA},
			readAllowed:  false,
			writeAllowed: false,
		},
	}

	for _, c := range cases {
		s.T().Run(c.title, func(t *testing.T) {
			forResourceHelpers := make([]sac.ForResourceHelper, 0, len(c.resources))
			for _, r := range c.resources {
				forResourceHelpers = append(forResourceHelpers, sac.ForResource(r))
			}
			forResources := sac.ForResources(forResourceHelpers...)

			read, err := forResources.ReadAllowedToAll(c.ctx)
			s.NoError(err)
			s.Equal(c.readAllowed, read)

			write, err := forResources.WriteAllowedToAll(c.ctx)
			s.NoError(err)
			s.Equal(c.writeAllowed, write)
		})
	}
}

func (s *forResourcesHelpersTestSuite) TestForAccessToAllWithScopeKeys() {
	cases := []struct {
		title        string
		ctx          context.Context
		resources    []permissions.ResourceMetadata
		scopeKeys    []sac.ScopeKey
		readAllowed  bool
		writeAllowed bool
	}{
		{
			title:        "Can only read when user has read on all the required resources with global scope",
			ctx:          readOnAllRes,
			resources:    []permissions.ResourceMetadata{resA, resB},
			scopeKeys:    sac.GlobalScopeKey(),
			readAllowed:  true,
			writeAllowed: false,
		},
		{
			title:        "Can only read when user has read on all the required resources with correct scope",
			ctx:          readOnAllScopedResWithCorrectScope,
			resources:    []permissions.ResourceMetadata{scopedRes, scopedResB},
			scopeKeys:    []sac.ScopeKey{sac.ClusterScopeKey("cluster-1")},
			readAllowed:  true,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has read on some of the required resources with correct scope",
			ctx:          readOnAScopedResWithCorrectScope,
			resources:    []permissions.ResourceMetadata{scopedRes, scopedResB},
			scopeKeys:    []sac.ScopeKey{sac.ClusterScopeKey("cluster-1")},
			readAllowed:  false,
			writeAllowed: false,
		},
		{
			title:        "Can't read or write when user has read on all the required resources with incorrect scope",
			ctx:          readOnAScopedResWithCorrectScope,
			resources:    []permissions.ResourceMetadata{scopedRes, scopedResB},
			scopeKeys:    []sac.ScopeKey{sac.ClusterScopeKey("cluster-2")},
			readAllowed:  false,
			writeAllowed: false,
		},
	}

	for _, c := range cases {
		s.T().Run(c.title, func(t *testing.T) {
			forResourceHelpers := make([]sac.ForResourceHelper, 0, len(c.resources))
			for _, r := range c.resources {
				forResourceHelpers = append(forResourceHelpers, sac.ForResource(r))
			}
			forResources := sac.ForResources(forResourceHelpers...)

			read, err := forResources.ReadAllowedToAll(c.ctx, c.scopeKeys...)
			s.NoError(err)
			s.Equal(c.readAllowed, read)

			write, err := forResources.WriteAllowedToAll(c.ctx, c.scopeKeys...)
			s.NoError(err)
			s.Equal(c.writeAllowed, write)
		})
	}
}
