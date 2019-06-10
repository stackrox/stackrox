package search

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

type searcherSuite struct {
	suite.Suite

	noAccessCtx       context.Context
	ns1ReadAccessCtx  context.Context
	fullReadAccessCtx context.Context

	store    store.Store
	indexer  index.Indexer
	searcher Searcher
}

func TestSearcher(t *testing.T) {
	suite.Run(t, new(searcherSuite))
}

func (s *searcherSuite) SetupSuite() {
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.DenyAllAccessScopeChecker())
	s.ns1ReadAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
			sac.ClusterScopeKeys("clusterA"),
			sac.NamespaceScopeKeys("ns1")))
	s.fullReadAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image)))
}

func (s *searcherSuite) SetupTest() {
	bleveIndex, err := globalindex.MemOnlyIndex()
	s.Require().NoError(err)

	s.indexer = index.New(bleveIndex)

	db, err := bolthelper.NewTemp(testutils.DBFileName(s))
	s.Require().NoError(err)

	s.store = store.New(db, false)

	s.searcher = New(s.store, s.indexer)
}

func (s *searcherSuite) TestNoAccess() {
	img := &storage.Image{
		Id: "img1",
		ClusternsScopes: map[string]string{
			"deploy1": sac.ClusterNSScopeString("clusterA", "ns2"),
			"deploy2": sac.ClusterNSScopeString("clusterB", "ns1"),
		},
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

	results, err = s.searcher.Search(s.ns1ReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

	results, err = s.searcher.Search(s.fullReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
}

func (s *searcherSuite) TestHasAccess() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().SkipNow()
	}

	img := &storage.Image{
		Id: "img1",
		ClusternsScopes: map[string]string{
			"deploy1": sac.ClusterNSScopeString("clusterA", "ns1"),
			"deploy2": sac.ClusterNSScopeString("clusterB", "ns2"),
		},
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

	results, err = s.searcher.Search(s.ns1ReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	results, err = s.searcher.Search(s.fullReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
}

func (s *searcherSuite) TestNoClusterNSScopes() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().SkipNow()
	}

	img := &storage.Image{
		Id: "img1",
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

	results, err = s.searcher.Search(s.ns1ReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

	results, err = s.searcher.Search(s.fullReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
}
