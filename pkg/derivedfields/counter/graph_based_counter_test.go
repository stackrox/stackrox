package counter

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stretchr/testify/suite"
)

var (
	prefix1       = []byte("pre1")
	prefix2       = []byte("pre2")
	prefix3       = []byte("pre3")
	clusterPrefix = []byte("cluster")

	id1  = []byte("id1")
	id2  = []byte("id2")
	id3  = []byte("id3")
	id4  = []byte("id4")
	id5  = []byte("id5")
	id6  = []byte("id6")
	id11 = []byte("id11")

	prefixedID1  = dbhelper.GetBucketKey(prefix1, id1)
	prefixedID2  = dbhelper.GetBucketKey(prefix2, id2)
	prefixedID3  = dbhelper.GetBucketKey(prefix2, id3)
	prefixedID4  = dbhelper.GetBucketKey(prefix3, id4)
	prefixedID5  = dbhelper.GetBucketKey(prefix3, id5)
	prefixedID6  = dbhelper.GetBucketKey(clusterPrefix, id6)
	prefixedID11 = dbhelper.GetBucketKey(prefix1, id11)

	// Fake hierarchy for test, use prefixed values since that is what will be stored in the graph.
	fromID1  = [][]byte{prefixedID2, prefixedID3}
	fromID2  = [][]byte{prefixedID4}
	fromID3  = [][]byte{prefixedID5}
	fromID11 = [][]byte{prefixedID2}

	globalResource = permissions.ResourceMetadata{
		Resource: "resource",
		Scope:    permissions.GlobalScope,
	}
)

func TestDerivedFieldCounter(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(derivedFieldCounterTestSuite))
}

type derivedFieldCounterTestSuite struct {
	suite.Suite

	mockRGraph *mocks.MockRGraph

	mockCtrl *gomock.Controller
}

func (s *derivedFieldCounterTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockRGraph = mocks.NewMockRGraph(s.mockCtrl)
}

func (s *derivedFieldCounterTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

/*
id1 -> id2 -> id4
id1 -> id3 -> id5
*/
func (s *derivedFieldCounterTestSuite) TestCounterForward() {
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID1).Return(fromID1)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID2).Return(fromID2)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID3).Return(fromID3)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
		filtered.WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	prefixPath := dackbox.Path{Path: [][]byte{prefix1, prefix2, prefix3}, ForwardTraversal: true}
	counter := NewGraphBasedDerivedFieldCounter(fakeGraphProvider{mg: s.mockRGraph}, prefixPath, filter)
	count, _ := counter.Count(ctx, string(id1))
	s.Equal(map[string]int32{string(id1): int32(2)}, count)
}

/*
id1 -> id2 -> id4
id1 -> id3
*/
func (s *derivedFieldCounterTestSuite) TestCounterForwardWithPartialPath() {
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID1).Return(fromID1)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID2).Return(fromID2)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID3).Return([][]byte{})

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
		filtered.WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	prefixPath := dackbox.Path{Path: [][]byte{prefix1, prefix2, prefix3}, ForwardTraversal: true}
	counter := NewGraphBasedDerivedFieldCounter(fakeGraphProvider{mg: s.mockRGraph}, prefixPath, filter)
	count, _ := counter.Count(ctx, string(id1))
	s.Equal(map[string]int32{string(id1): int32(1)}, count)
}

/*
id1 -> id2 -> id5
id1 -> id3 -> id5
*/
func (s *derivedFieldCounterTestSuite) TestCounterForwardRepeated() {
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID1).Return(fromID1)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID2).Return([][]byte{prefixedID5})
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID3).Return(fromID3)

	graphProvider := fakeGraphProvider{mg: s.mockRGraph}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
		filtered.WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	prefixPath := dackbox.Path{Path: [][]byte{prefix1, prefix2, prefix3}, ForwardTraversal: true}
	counter := NewGraphBasedDerivedFieldCounter(graphProvider, prefixPath, filter)
	count, _ := counter.Count(ctx, string(id1))
	s.Equal(map[string]int32{string(id1): int32(1)}, count)
}

/*
id1 -> id2 -> id4
id1 -> id3 -> id5
id1 -> id3 -> id6
id11 -> id2 ->id4
*/
func (s *derivedFieldCounterTestSuite) TestCounterForwardOneToMany() {
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID1).Return(fromID1)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID2).Return(fromID2)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID3).Return([][]byte{prefixedID5, dbhelper.GetBucketKey(prefix3, id6)})
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID11).Return(fromID11)

	graphProvider := fakeGraphProvider{mg: s.mockRGraph}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
		filtered.WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	prefixPath := dackbox.Path{Path: [][]byte{prefix1, prefix2, prefix3}, ForwardTraversal: true}
	counter := NewGraphBasedDerivedFieldCounter(graphProvider, prefixPath, filter)
	count, _ := counter.Count(ctx, string(id1), string(id11))
	s.Equal(map[string]int32{string(id1): int32(3), string(id11): int32(1)}, count)
}

/*
id1 -> id2 -> id4
id1 -> id3 -> id5
id1 -> id3 -> id6 (diff prefix)
*/
func (s *derivedFieldCounterTestSuite) TestCounterForwardWithDiffPrefix() {
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID1).Return(fromID1)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID2).Return(fromID2)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID3).Return([][]byte{prefixedID5, prefixedID6})

	graphProvider := fakeGraphProvider{mg: s.mockRGraph}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
		filtered.WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	prefixPath := dackbox.Path{Path: [][]byte{prefix1, prefix2, prefix3}, ForwardTraversal: true}
	counter := NewGraphBasedDerivedFieldCounter(graphProvider, prefixPath, filter)
	count, _ := counter.Count(ctx, string(id1))
	s.Equal(map[string]int32{string(id1): int32(2)}, count)
}

type fakeGraphProvider struct {
	mg *mocks.MockRGraph
}

func (fgp fakeGraphProvider) NewGraphView() graph.DiscardableRGraph {
	return graph.NewDiscardableGraph(fgp.mg, func() {})
}
