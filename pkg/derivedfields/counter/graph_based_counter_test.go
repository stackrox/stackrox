package counter

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/stackrox/pkg/dbhelper"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	"github.com/stretchr/testify/suite"
)

var (
	prefix1 = []byte("pre1")
	prefix2 = []byte("pre2")
	prefix3 = []byte("pre3")

	id1  = []byte("id1")
	id2  = []byte("id2")
	id3  = []byte("id3")
	id4  = []byte("id4")
	id5  = []byte("id5")
	id6  = []byte("id6")
	id11 = []byte("id11")

	prefixed1ID1  = dbhelper.GetBucketKey(prefix1, id1)
	prefixed2ID2  = dbhelper.GetBucketKey(prefix2, id2)
	prefixed2ID3  = dbhelper.GetBucketKey(prefix2, id3)
	prefixed3ID4  = dbhelper.GetBucketKey(prefix3, id4)
	prefixed3ID5  = dbhelper.GetBucketKey(prefix3, id5)
	prefixed1ID11 = dbhelper.GetBucketKey(prefix1, id11)

	// Fake hierarchy for test, use prefixed values since that is what will be stored in the graph.
	fromID1Prefixed2  = [][]byte{prefixed2ID2, prefixed2ID3}
	fromID2Prefixed3  = [][]byte{prefixed3ID4}
	fromID3Prefixed3  = [][]byte{prefixed3ID5}
	fromID11Prefixed2 = [][]byte{prefixed2ID2}

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
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed1ID1, prefix2).Return(fromID1Prefixed2)
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID2, prefix3).Return(fromID2Prefixed3)
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID3, prefix3).Return(fromID3Prefixed3)

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
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed1ID1, prefix2).Return(fromID1Prefixed2)
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID2, prefix3).Return(fromID2Prefixed3)
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID3, prefix3).Return([][]byte{})

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
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed1ID1, prefix2).Return(fromID1Prefixed2)
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID2, prefix3).Return([][]byte{prefixed3ID5})
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID3, prefix3).Return(fromID3Prefixed3)

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
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed1ID1, prefix2).Return(fromID1Prefixed2)
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID2, prefix3).Return(fromID2Prefixed3)
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID3, prefix3).Return([][]byte{prefixed3ID5, dbhelper.GetBucketKey(prefix3, id6)})
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed1ID11, prefix2).Return(fromID11Prefixed2)

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
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed1ID1, prefix2).Return(fromID1Prefixed2)
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID2, prefix3).Return(fromID2Prefixed3)
	s.mockRGraph.EXPECT().GetRefsFromPrefix(prefixed2ID3, prefix3).Return([][]byte{prefixed3ID5})

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
