package idspace

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stretchr/testify/suite"
)

var (
	prefix1 = []byte("pre1")
	prefix2 = []byte("pre2")
	prefix3 = []byte("pre3")

	id1 = []byte("id1")
	id2 = []byte("id2")
	id3 = []byte("id3")
	id4 = []byte("id4")
	id5 = []byte("id5")

	prefixedID1 = badgerhelper.GetBucketKey(prefix1, []byte("id1"))
	prefixedID2 = badgerhelper.GetBucketKey(prefix2, []byte("id2"))
	prefixedID3 = badgerhelper.GetBucketKey(prefix2, []byte("id3"))
	prefixedID4 = badgerhelper.GetBucketKey(prefix3, []byte("id4"))
	prefixedID5 = badgerhelper.GetBucketKey(prefix3, []byte("id5"))

	// Fake hierarchy for test, use prefixed values since that is what will be stored in the graph.
	fromID1 = [][]byte{prefixedID2, prefixedID3}
	fromID2 = [][]byte{prefixedID4}
	fromID3 = [][]byte{prefixedID5}

	toID2 = [][]byte{prefixedID1}
	toID3 = [][]byte{prefixedID1}
	toID4 = [][]byte{prefixedID2}
	toID5 = [][]byte{prefixedID3}
)

func TestIDTransformation(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(idTransformationTestSuite))
}

type idTransformationTestSuite struct {
	suite.Suite

	mockRGraph *mocks.MockRGraph

	mockCtrl *gomock.Controller
}

func (s *idTransformationTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockRGraph = mocks.NewMockRGraph(s.mockCtrl)
}

func (s *idTransformationTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *idTransformationTestSuite) TestTransformForward() {
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID1).Return(fromID1)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID2).Return(fromID2)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID3).Return(fromID3)

	transformer := NewForwardGraphTransformer(fakeGraphProvider{mg: s.mockRGraph}, [][]byte{prefix1, prefix2, prefix3})
	transformed, _ := transformer.Transform(string(id1))
	s.Equal([]string{string(id4), string(id5)}, transformed)
}

func (s *idTransformationTestSuite) TestTransformForwardBatch() {
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID2).Return(fromID2)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID3).Return(fromID3)

	transformer := NewForwardGraphTransformer(fakeGraphProvider{mg: s.mockRGraph}, [][]byte{prefix2, prefix3})
	transformed, _ := transformer.Transform(string(id2), string(id3))
	s.Equal([]string{string(id4), string(id5)}, transformed)
}

func (s *idTransformationTestSuite) TestTransformBackward() {
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)

	transformer := NewBackwardGraphTransformer(fakeGraphProvider{mg: s.mockRGraph}, [][]byte{prefix3, prefix2, prefix1})
	transformed, _ := transformer.Transform(string(id4))
	s.Equal([]string{string(id1)}, transformed)
}

func (s *idTransformationTestSuite) TestTransformBackwardBatch() {
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID5).Return(toID5)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID3).Return(toID3)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)

	transformer := NewBackwardGraphTransformer(fakeGraphProvider{mg: s.mockRGraph}, [][]byte{prefix3, prefix2, prefix1})
	transformed, _ := transformer.Transform(string(id4), string(id5))
	s.Equal([]string{string(id1)}, transformed)
}

func (s *idTransformationTestSuite) TestTransformForwardMultiplePrefixPaths() {
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID1).Return(fromID1)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID2).Return(fromID2)
	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID3).Return(fromID3)

	s.mockRGraph.EXPECT().GetRefsFrom(prefixedID1).Return(fromID1)

	transformer := NewForwardGraphTransformer(fakeGraphProvider{mg: s.mockRGraph}, [][]byte{prefix1, prefix2, prefix3}, [][]byte{prefix1, prefix2})
	transformed, _ := transformer.Transform(string(id1))
	s.Equal([]string{string(id2), string(id3), string(id4), string(id5)}, transformed)
}

func (s *idTransformationTestSuite) TestTransformBackwardMultiplePrefixPaths() {
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)

	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)

	transformer := NewBackwardGraphTransformer(fakeGraphProvider{mg: s.mockRGraph}, [][]byte{prefix3, prefix2, prefix1}, [][]byte{prefix3, prefix2})
	transformed, _ := transformer.Transform(string(id4))
	s.Equal([]string{string(id1), string(id2)}, transformed)
}

type fakeGraphProvider struct {
	mg *mocks.MockRGraph
}

func (fgp fakeGraphProvider) NewGraphView() graph.DiscardableRGraph {
	return graph.NewDiscardableGraph(fgp.mg, func() {})
}
