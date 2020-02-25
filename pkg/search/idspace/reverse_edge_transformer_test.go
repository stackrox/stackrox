package idspace

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stretchr/testify/suite"
)

var (
	id11 = []byte("id11")
	id12 = []byte("id12")

	prefixedID11 = badgerhelper.GetBucketKey(prefix2, []byte("id11"))
	prefixedID12 = badgerhelper.GetBucketKey(prefix2, []byte("id12"))

	toID11 = [][]byte{prefixedID1}
	toID12 = [][]byte{prefixedID1}
)

func TestReverseEdgeTransformation(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(reverseEdgeTransformationTestSuite))
}

type reverseEdgeTransformationTestSuite struct {
	suite.Suite

	mockRGraph *mocks.MockRGraph

	mockCtrl *gomock.Controller
}

func (s *reverseEdgeTransformationTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockRGraph = mocks.NewMockRGraph(s.mockCtrl)
}

func (s *reverseEdgeTransformationTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *reverseEdgeTransformationTestSuite) TestTransformParentToEdge() {
	expectedEdges := []string{
		edges.EdgeID{ParentID: "id1", ChildID: "id2"}.ToString(),
		edges.EdgeID{ParentID: "id1", ChildID: "id3"}.ToString(),
	}

	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID1).Return(fromID1)

	transformer := NewParentToEdgeTransformer(fakeGraphProvider{mg: s.mockRGraph}, [][]byte{prefix1, prefix2})
	transformed, _ := transformer.Transform(string(id1))
	s.Equal(expectedEdges, transformed)
}

func (s *reverseEdgeTransformationTestSuite) TestTransformChildToEdge() {
	expectedEdges := []string{
		edges.EdgeID{ParentID: "id1", ChildID: "id2"}.ToString(),
	}

	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)

	transformer := NewChildToEdgeTransformer(fakeGraphProvider{mg: s.mockRGraph}, [][]byte{prefix2, prefix1})
	transformed, _ := transformer.Transform(string(id2))
	s.Equal(expectedEdges, transformed)
}

func (s *reverseEdgeTransformationTestSuite) TestTransformManyChildToEdge() {
	expectedEdges := []string{
		edges.EdgeID{ParentID: "id1", ChildID: "id2"}.ToString(),
		edges.EdgeID{ParentID: "id1", ChildID: "id3"}.ToString(),
		edges.EdgeID{ParentID: "id1", ChildID: "id11"}.ToString(),
		edges.EdgeID{ParentID: "id1", ChildID: "id12"}.ToString(),
	}

	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID3).Return(toID3)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID11).Return(toID11)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID12).Return(toID12)

	transformer := NewChildToEdgeTransformer(fakeGraphProvider{mg: s.mockRGraph}, [][]byte{prefix2, prefix1})
	transformed, _ := transformer.Transform(string(id2), string(id3), string(id11), string(id12))
	s.Equal(expectedEdges, transformed)
}
