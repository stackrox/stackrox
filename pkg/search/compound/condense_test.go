package compound

import (
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
	searchMocks "github.com/stackrox/stackrox/pkg/search/mocks"
	"github.com/stretchr/testify/suite"
)

func TestCondenseRequest(t *testing.T) {
	suite.Run(t, new(CondenseRequestTestSuite))
}

type CondenseRequestTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	mockSearcher1 *searchMocks.MockSearcher
	mockSearcher2 *searchMocks.MockSearcher
}

func (suite *CondenseRequestTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.mockSearcher1 = searchMocks.NewMockSearcher(suite.mockCtrl)
	suite.mockSearcher2 = searchMocks.NewMockSearcher(suite.mockCtrl)
}

func (suite *CondenseRequestTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *CondenseRequestTestSuite) TestCondenseAnd() {
	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
		},
		{
			Searcher: suite.mockSearcher2,
		},
	}

	input := &searchRequestSpec{
		and: []*searchRequestSpec{
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
		},
	}

	expected := &searchRequestSpec{
		base: &baseRequestSpec{
			Spec: &searcherSpecs[0],
			Query: search.ConjunctionQuery(
				&v1.Query{},
				&v1.Query{},
			),
		},
	}

	actual, err := condense(input)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *CondenseRequestTestSuite) TestCondenseOr() {
	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
		},
		{
			Searcher: suite.mockSearcher2,
		},
	}

	input := &searchRequestSpec{
		or: []*searchRequestSpec{
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
		},
	}

	expected := &searchRequestSpec{
		base: &baseRequestSpec{
			Spec: &searcherSpecs[0],
			Query: search.DisjunctionQuery(
				&v1.Query{},
				&v1.Query{},
			),
		},
	}

	actual, err := condense(input)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *CondenseRequestTestSuite) TestCondenseBoolean() {
	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
		},
		{
			Searcher: suite.mockSearcher2,
		},
	}

	input := &searchRequestSpec{
		boolean: &booleanRequestSpec{
			must: &searchRequestSpec{
				and: []*searchRequestSpec{
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[0],
							Query: &v1.Query{},
						},
					},
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[0],
							Query: &v1.Query{},
						},
					},
				},
			},
			mustNot: &searchRequestSpec{
				or: []*searchRequestSpec{
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[1],
							Query: &v1.Query{},
						},
					},
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[1],
							Query: &v1.Query{},
						},
					},
				},
			},
		},
	}

	expected := &searchRequestSpec{
		boolean: &booleanRequestSpec{
			must: &searchRequestSpec{
				base: &baseRequestSpec{
					Spec: &searcherSpecs[0],
					Query: search.ConjunctionQuery(
						&v1.Query{},
						&v1.Query{},
					),
				},
			},
			mustNot: &searchRequestSpec{
				base: &baseRequestSpec{
					Spec: &searcherSpecs[1],
					Query: search.DisjunctionQuery(
						&v1.Query{},
						&v1.Query{},
					),
				},
			},
		},
	}

	actual, err := condense(input)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *CondenseRequestTestSuite) TestNotCondensable() {
	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
		},
		{
			Searcher: suite.mockSearcher2,
		},
	}

	expectedQ1 := search.NewQueryBuilder().
		AddLinkedFieldsWithHighlightValues(
			[]search.FieldLabel{"s1field", "s3field"},
			[]string{"s1value", "s3value"},
			[]bool{true, true}).
		ProtoQuery()

	expectedQ3 := search.NewQueryBuilder().
		AddExactMatches("s3field", "s3value").
		ProtoQuery()

	expectedQ4 := search.NewQueryBuilder().
		AddExactMatches("s4field", "s4value").
		ProtoQuery()

	expected := &searchRequestSpec{
		or: []*searchRequestSpec{
			{
				and: []*searchRequestSpec{
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[0],
							Query: expectedQ3,
						},
					},
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[1],
							Query: expectedQ4,
						},
					},
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: expectedQ1,
				},
			},
		},
	}

	actual, err := condense(expected)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *CondenseRequestTestSuite) TestCondenseComplex() {
	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
		},
		{
			Searcher: suite.mockSearcher2,
		},
	}

	input := &searchRequestSpec{
		and: []*searchRequestSpec{
			{
				boolean: &booleanRequestSpec{
					must: &searchRequestSpec{
						and: []*searchRequestSpec{
							{
								base: &baseRequestSpec{
									Spec:  &searcherSpecs[0],
									Query: &v1.Query{},
								},
							},
							{
								base: &baseRequestSpec{
									Spec:  &searcherSpecs[0],
									Query: &v1.Query{},
								},
							},
						},
					},
					mustNot: &searchRequestSpec{
						or: []*searchRequestSpec{
							{
								base: &baseRequestSpec{
									Spec:  &searcherSpecs[1],
									Query: &v1.Query{},
								},
							},
							{
								base: &baseRequestSpec{
									Spec:  &searcherSpecs[0],
									Query: &v1.Query{},
								},
							},
						},
					},
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
			{
				or: []*searchRequestSpec{
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[0],
							Query: &v1.Query{},
						},
					},
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[0],
							Query: &v1.Query{},
						},
					},
				},
			},
		},
	}

	expected := &searchRequestSpec{
		and: []*searchRequestSpec{
			{
				boolean: &booleanRequestSpec{
					must: &searchRequestSpec{
						base: &baseRequestSpec{
							Spec: &searcherSpecs[0],
							Query: search.ConjunctionQuery(
								&v1.Query{},
								&v1.Query{},
							),
						},
					},
					mustNot: &searchRequestSpec{
						or: []*searchRequestSpec{
							{
								base: &baseRequestSpec{
									Spec:  &searcherSpecs[1],
									Query: &v1.Query{},
								},
							},
							{
								base: &baseRequestSpec{
									Spec:  &searcherSpecs[0],
									Query: &v1.Query{},
								},
							},
						},
					},
				},
			},
			{
				base: &baseRequestSpec{
					Spec: &searcherSpecs[0],
					Query: search.ConjunctionQuery(
						&v1.Query{},
						&v1.Query{},
						search.DisjunctionQuery(
							&v1.Query{},
							&v1.Query{},
						),
					),
				},
			},
		},
	}

	actual, err := condense(input)
	suite.Nil(err)
	suite.Equal(expected, actual)
}
