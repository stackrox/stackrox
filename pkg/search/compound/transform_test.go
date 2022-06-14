package compound

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/mocks"
	"github.com/stretchr/testify/suite"
)

func TestWithIDTransformation(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(idTransformedSearcherTestSuite))
}

type idTransformedSearcherTestSuite struct {
	suite.Suite

	mockSearcher *mocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (s *idTransformedSearcherTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockSearcher = mocks.NewMockSearcher(s.mockCtrl)
}

func (s *idTransformedSearcherTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *idTransformedSearcherTestSuite) TestTransformParentToEdge() {
	results := []search.Result{
		{
			ID: "1",
			Matches: map[string][]string{
				"1:1": {"a", "b"},
			},
		},
		{
			ID: "2",
			Matches: map[string][]string{
				"2:1": {"c", "d"},
				"2:2": {"e", "f"},
			},
		},
		{
			ID: "3",
			Matches: map[string][]string{
				"1:1": {"g", "h"},
			},
		},
	}

	expectedResults := []search.Result{
		{
			ID: "4",
			Matches: map[string][]string{
				"1:1": {"a", "b"},
				"2:1": {"c", "d"},
				"2:2": {"e", "f"},
			},
		},
		{
			ID: "5",
			Matches: map[string][]string{
				"1:1": {"a", "b", "g", "h"},
			},
		},
		{
			ID: "6",
			Matches: map[string][]string{
				"2:1": {"c", "d"},
				"2:2": {"e", "f"},
				"1:1": {"g", "h"},
			},
		},
	}

	fakeTransform := func(_ context.Context, key []byte) [][]byte {
		sKey := string(key)
		if sKey == "1" {
			return [][]byte{[]byte("4"), []byte("5")}
		} else if sKey == "2" {
			return [][]byte{[]byte("4"), []byte("6")}
		} else {
			return [][]byte{[]byte("5"), []byte("6")}
		}
	}

	transformedResults := TransformResults(context.Background(), results, fakeTransform)
	s.Equal(expectedResults, transformedResults)
}
