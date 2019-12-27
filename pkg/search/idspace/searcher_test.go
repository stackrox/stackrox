package idspace

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/mocks"
	"github.com/stretchr/testify/suite"
)

func TestIdTransformation(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(idTransformationTestSuite))
}

type idTransformationTestSuite struct {
	suite.Suite

	mockSearcher *mocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (s *idTransformationTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockSearcher = mocks.NewMockSearcher(s.mockCtrl)
}

func (s *idTransformationTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *idTransformationTestSuite) TestHandlesNoIdTransformation() {
	s.mockSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return(fakeResults, nil)

	results, err := TransformIDs(s.mockSearcher, EdgeIDToParentID).Search(context.Background(), &v1.Query{})
	s.NoError(err, "expected no error, should return nil without access")
	s.Equal(fakeResultParents, results)
}

func (s *idTransformationTestSuite) TestHandlesNoOffset() {
	s.mockSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return(fakeResults, nil)

	results, err := TransformIDs(s.mockSearcher, EdgeIDToChildID).Search(context.Background(), &v1.Query{})
	s.NoError(err, "expected no error, should return nil without access")
	s.Equal(fakeResultChildren, results)
}

var fakeResults = []search.Result{
	{
		ID: edges.EdgeID{ParentID: "parent1", ChildID: "child1"}.ToString(),
	},
	{
		ID: edges.EdgeID{ParentID: "parent2", ChildID: "child2"}.ToString(),
	},
	{
		ID: edges.EdgeID{ParentID: "parent3", ChildID: "child1"}.ToString(),
	},
	{
		ID: edges.EdgeID{ParentID: "parent3", ChildID: "child4"}.ToString(),
	},
	{
		ID: edges.EdgeID{ParentID: "parent5", ChildID: "child5"}.ToString(),
	},
}

var fakeResultParents = []search.Result{
	{
		ID: "parent1",
	},
	{
		ID: "parent2",
	},
	{
		ID: "parent3",
	},
	{
		ID: "parent5",
	},
}

var fakeResultChildren = []search.Result{
	{
		ID: "child1",
	},
	{
		ID: "child2",
	},
	{
		ID: "child4",
	},
	{
		ID: "child5",
	},
}
