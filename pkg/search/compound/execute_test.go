package compound

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
	searchMocks "github.com/stackrox/stackrox/pkg/search/mocks"
	"github.com/stretchr/testify/suite"
)

func TestRequestExecution(t *testing.T) {
	suite.Run(t, new(RequestExecutionTestSuite))
}

type RequestExecutionTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	mockOptions1  *searchMocks.MockOptionsMap
	mockOptions2  *searchMocks.MockOptionsMap
	mockSearcher1 *searchMocks.MockSearcher
	mockSearcher2 *searchMocks.MockSearcher
}

func (suite *RequestExecutionTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.mockOptions1 = searchMocks.NewMockOptionsMap(suite.mockCtrl)
	suite.mockOptions2 = searchMocks.NewMockOptionsMap(suite.mockCtrl)
	suite.mockSearcher1 = searchMocks.NewMockSearcher(suite.mockCtrl)
	suite.mockSearcher2 = searchMocks.NewMockSearcher(suite.mockCtrl)
}

func (suite *RequestExecutionTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *RequestExecutionTestSuite) TestExecuteBase() {
	ctx := context.Background()

	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
			Options:  suite.mockOptions1,
		},
	}

	q1 := &v1.Query{}

	testRequest := searchRequestSpec{
		base: &baseRequestSpec{
			Spec:  &searcherSpecs[0],
			Query: q1,
		},
	}

	testResult := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "3",
		},
	}

	suite.mockSearcher1.EXPECT().Search(ctx, q1).Return(testResult, nil)

	expected := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "3",
		},
	}

	actual, err := execute(ctx, &testRequest)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *RequestExecutionTestSuite) TestExecuteOr() {
	ctx := context.Background()

	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
			Options:  suite.mockOptions1,
		},
		{
			Searcher: suite.mockSearcher2,
			Options:  suite.mockOptions2,
		},
	}

	q1 := &v1.Query{}
	q2 := &v1.Query{}

	testRequest := searchRequestSpec{
		or: []*searchRequestSpec{
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: q1,
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[1],
					Query: q2,
				},
			},
		},
	}

	testResult1 := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "3",
		},
	}

	suite.mockSearcher1.EXPECT().Search(ctx, q1).Return(testResult1, nil)

	testResult2 := []search.Result{
		{
			ID: "5",
		},
		{
			ID: "1",
		},
		{
			ID: "9",
		},
	}

	suite.mockSearcher2.EXPECT().Search(ctx, q2).Return(testResult2, nil)

	expected := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "3",
		},
		{
			ID: "5",
		},
		{
			ID: "9",
		},
	}

	actual, err := execute(ctx, &testRequest)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *RequestExecutionTestSuite) TestExecuteAnd() {
	ctx := context.Background()

	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
			Options:  suite.mockOptions1,
		},
		{
			Searcher: suite.mockSearcher2,
			Options:  suite.mockOptions2,
		},
	}

	q1 := &v1.Query{}
	q2 := &v1.Query{}

	testRequest := searchRequestSpec{
		and: []*searchRequestSpec{
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: q1,
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[1],
					Query: q2,
				},
			},
		},
	}

	testResult1 := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "3",
		},
	}

	suite.mockSearcher1.EXPECT().Search(ctx, q1).Return(testResult1, nil)

	testResult2 := []search.Result{
		{
			ID: "5",
		},
		{
			ID: "1",
		},
		{
			ID: "9",
		},
	}

	suite.mockSearcher2.EXPECT().Search(ctx, q2).Return(testResult2, nil)

	expected := []search.Result{
		{
			ID: "1",
		},
	}

	actual, err := execute(ctx, &testRequest)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *RequestExecutionTestSuite) TestExecuteBoolean() {
	ctx := context.Background()

	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
			Options:  suite.mockOptions1,
		},
		{
			Searcher: suite.mockSearcher2,
			Options:  suite.mockOptions2,
		},
	}

	q1 := &v1.Query{}
	q2 := &v1.Query{}

	testRequest := searchRequestSpec{
		boolean: &booleanRequestSpec{
			must: &searchRequestSpec{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: q1,
				},
			},
			mustNot: &searchRequestSpec{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[1],
					Query: q2,
				},
			},
		},
	}

	testResult1 := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "3",
		},
	}

	suite.mockSearcher1.EXPECT().Search(ctx, q1).Return(testResult1, nil)

	testResult2 := []search.Result{
		{
			ID: "5",
		},
		{
			ID: "1",
		},
		{
			ID: "9",
		},
	}

	suite.mockSearcher2.EXPECT().Search(ctx, q2).Return(testResult2, nil)

	expected := []search.Result{
		{
			ID: "3",
		},
	}

	actual, err := execute(ctx, &testRequest)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *RequestExecutionTestSuite) TestExecuteLeftJoin() {
	ctx := context.Background()

	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
			Options:  suite.mockOptions1,
		},
		{
			Searcher: suite.mockSearcher2,
			Options:  suite.mockOptions2,
		},
	}

	q1 := &v1.Query{}
	q2 := &v1.Query{}

	testRequest := searchRequestSpec{
		leftJoinWithRightOrder: &joinRequestSpec{
			left: &searchRequestSpec{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: q1,
				},
			},
			right: &searchRequestSpec{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[1],
					Query: q2,
				},
			},
		},
	}

	testResult1 := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "3",
		},
		{
			ID: "6",
		},
	}

	suite.mockSearcher1.EXPECT().Search(ctx, q1).Return(testResult1, nil)

	testResult2 := []search.Result{
		{
			ID: "6",
		},
		{
			ID: "2",
		},
		{
			ID: "1",
		},
	}

	suite.mockSearcher2.EXPECT().Search(ctx, q2).Return(testResult2, nil)

	expected := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "3",
		},
		{
			ID: "6",
		},
	}

	actual, err := execute(ctx, &testRequest)
	suite.Nil(err)
	suite.Equal(expected, actual)
}
