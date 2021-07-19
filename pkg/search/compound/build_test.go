package compound

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/search"
	searchMocks "github.com/stackrox/rox/pkg/search/mocks"
	"github.com/stretchr/testify/suite"
)

func TestBuildRequest(t *testing.T) {
	suite.Run(t, new(BuildRequestTestSuite))
}

type BuildRequestTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	mockOptions1  *searchMocks.MockOptionsMap
	mockOptions2  *searchMocks.MockOptionsMap
	mockSearcher1 *searchMocks.MockSearcher
	mockSearcher2 *searchMocks.MockSearcher
}

func (suite *BuildRequestTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.mockOptions1 = searchMocks.NewMockOptionsMap(suite.mockCtrl)
	suite.mockOptions2 = searchMocks.NewMockOptionsMap(suite.mockCtrl)
	suite.mockSearcher1 = searchMocks.NewMockSearcher(suite.mockCtrl)
	suite.mockSearcher2 = searchMocks.NewMockSearcher(suite.mockCtrl)
}

func (suite *BuildRequestTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *BuildRequestTestSuite) TestBuildDefault() {
	searcherSpecs := []SearcherSpec{
		{
			Searcher: suite.mockSearcher1,
			Options:  suite.mockOptions1,
		},
		{
			IsDefault: true,
			Searcher:  suite.mockSearcher2,
			Options:   suite.mockOptions2,
		},
	}

	q1 := search.EmptyQuery()

	expected := &searchRequestSpec{
		base: &baseRequestSpec{
			Spec:  &searcherSpecs[1],
			Query: search.EmptyQuery(),
		},
	}

	actual, err := build(q1, searcherSpecs)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *BuildRequestTestSuite) TestBuildBase() {
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

	q1 := search.NewQueryBuilder().
		AddExactMatches("s1field", "s1value").
		AddExactMatches("s2field", "s2value").
		ProtoQuery()

	suite.mockOptions1.EXPECT().Get("s1field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s2field").Return(nil, false)
	suite.mockOptions2.EXPECT().Get("s2field").Return(nil, true)

	expectedQ1 := search.NewQueryBuilder().
		AddExactMatches("s1field", "s1value").
		ProtoQuery()

	expectedQ2 := search.NewQueryBuilder().
		AddExactMatches("s2field", "s2value").
		ProtoQuery()

	expected := &searchRequestSpec{
		and: []*searchRequestSpec{
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: expectedQ1,
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[1],
					Query: expectedQ2,
				},
			},
		},
	}

	actual, err := build(q1, searcherSpecs)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *BuildRequestTestSuite) TestBuildConjunction() {
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

	q1 := search.ConjunctionQuery(
		search.NewQueryBuilder().
			AddExactMatches("s1field", "s1value").
			AddExactMatches("s2field", "s2value").
			ProtoQuery(),
		search.NewQueryBuilder().
			AddExactMatches("s3field", "s3value").
			AddExactMatches("s4field", "s4value").
			ProtoQuery(),
	)

	suite.mockOptions1.EXPECT().Get("s1field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s2field").Return(nil, false)
	suite.mockOptions1.EXPECT().Get("s3field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s4field").Return(nil, false)
	suite.mockOptions2.EXPECT().Get("s2field").Return(nil, true)
	suite.mockOptions2.EXPECT().Get("s4field").Return(nil, true)

	expectedQ1 := search.NewQueryBuilder().
		AddExactMatches("s1field", "s1value").
		ProtoQuery()

	expectedQ2 := search.NewQueryBuilder().
		AddExactMatches("s2field", "s2value").
		ProtoQuery()

	expectedQ3 := search.NewQueryBuilder().
		AddExactMatches("s3field", "s3value").
		ProtoQuery()

	expectedQ4 := search.NewQueryBuilder().
		AddExactMatches("s4field", "s4value").
		ProtoQuery()

	expected := &searchRequestSpec{
		and: []*searchRequestSpec{
			{
				and: []*searchRequestSpec{
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[0],
							Query: expectedQ1,
						},
					},
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[1],
							Query: expectedQ2,
						},
					},
				},
			},
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
		},
	}

	actual, err := build(q1, searcherSpecs)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *BuildRequestTestSuite) TestBuildDisjunction() {
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

	q1 := search.DisjunctionQuery(
		search.NewQueryBuilder().
			AddExactMatches("s1field", "s1value").
			AddExactMatches("s2field", "s2value").
			ProtoQuery(),
		search.NewQueryBuilder().
			AddExactMatches("s3field", "s3value").
			AddExactMatches("s4field", "s4value").
			ProtoQuery(),
	)

	suite.mockOptions1.EXPECT().Get("s1field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s2field").Return(nil, false)
	suite.mockOptions1.EXPECT().Get("s3field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s4field").Return(nil, false)
	suite.mockOptions2.EXPECT().Get("s2field").Return(nil, true)
	suite.mockOptions2.EXPECT().Get("s4field").Return(nil, true)

	expectedQ1 := search.NewQueryBuilder().
		AddExactMatches("s1field", "s1value").
		ProtoQuery()

	expectedQ2 := search.NewQueryBuilder().
		AddExactMatches("s2field", "s2value").
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
							Query: expectedQ1,
						},
					},
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[1],
							Query: expectedQ2,
						},
					},
				},
			},
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
		},
	}

	actual, err := build(q1, searcherSpecs)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *BuildRequestTestSuite) TestBuildLinkedDisjunction() {
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

	q1 := search.DisjunctionQuery(
		search.NewQueryBuilder().
			AddLinkedFieldsWithHighlightValues(
				[]search.FieldLabel{"s1field", "s3field"},
				[]string{"s1value", "s3value"},
				[]bool{true, true}).
			ProtoQuery(),
		search.NewQueryBuilder().
			AddExactMatches("s3field", "s3value").
			AddExactMatches("s4field", "s4value").
			ProtoQuery(),
	)

	suite.mockOptions1.EXPECT().Get("s1field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s3field").Return(nil, true)

	suite.mockOptions1.EXPECT().Get("s3field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s4field").Return(nil, false)
	suite.mockOptions2.EXPECT().Get("s4field").Return(nil, true)

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
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: expectedQ1,
				},
			},
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
		},
	}

	actual, err := build(q1, searcherSpecs)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *BuildRequestTestSuite) TestBuildBoolean() {
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

	q1 := search.NewBooleanQuery(
		search.ConjunctionQuery(
			search.NewQueryBuilder().AddExactMatches("s1field", "s1value").ProtoQuery(),
			search.NewQueryBuilder().AddExactMatches("s2field", "s2value").ProtoQuery(),
		).GetConjunction(),
		search.DisjunctionQuery(
			search.NewQueryBuilder().AddExactMatches("s3field", "s3value").ProtoQuery(),
			search.NewQueryBuilder().AddExactMatches("s4field", "s4value").ProtoQuery(),
		).GetDisjunction(),
	)

	suite.mockOptions1.EXPECT().Get("s1field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s2field").Return(nil, false)
	suite.mockOptions1.EXPECT().Get("s3field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s4field").Return(nil, false)
	suite.mockOptions2.EXPECT().Get("s2field").Return(nil, true)
	suite.mockOptions2.EXPECT().Get("s4field").Return(nil, true)

	expectedQ1 := search.NewQueryBuilder().
		AddExactMatches("s1field", "s1value").
		ProtoQuery()

	expectedQ2 := search.NewQueryBuilder().
		AddExactMatches("s2field", "s2value").
		ProtoQuery()

	expectedQ3 := search.NewQueryBuilder().
		AddExactMatches("s3field", "s3value").
		ProtoQuery()

	expectedQ4 := search.NewQueryBuilder().
		AddExactMatches("s4field", "s4value").
		ProtoQuery()

	expected := &searchRequestSpec{
		boolean: &booleanRequestSpec{
			must: &searchRequestSpec{
				and: []*searchRequestSpec{
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[0],
							Query: expectedQ1,
						},
					},
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[1],
							Query: expectedQ2,
						},
					},
				},
			},
			mustNot: &searchRequestSpec{
				or: []*searchRequestSpec{
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
		},
	}

	actual, err := build(q1, searcherSpecs)
	suite.Nil(err)
	suite.Equal(expected, actual)
}

func (suite *BuildRequestTestSuite) TestBuildSingleLinked() {
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

	q1 := search.NewQueryBuilder().AddLinkedFields(
		[]search.FieldLabel{"s1field", "s2field"},
		[]string{"s1value", "s2value"},
	).ProtoQuery()

	// Look for a searcher with all linked fields.
	suite.mockOptions1.EXPECT().Get("s1field").Return(nil, true)
	suite.mockOptions1.EXPECT().Get("s2field").Return(nil, false)
	suite.mockOptions2.EXPECT().Get("s1field").Return(nil, false)

	// Fall back to allowing linked from the first with a match.
	suite.mockOptions2.EXPECT().Get("s1field").Return(nil, true)

	expectedQuery := search.ConjunctionQuery(
		search.NewQueryBuilder().
			AddStrings("s1field", "s1value").
			ProtoQuery(),
		search.NewQueryBuilder().
			AddStrings("s2field", "s2value").
			ProtoQuery(),
	)

	actual, err := build(q1, searcherSpecs)
	suite.Nil(err)
	suite.NotNil(actual.base)
	suite.Equal(expectedQuery, actual.base.Query)
}
