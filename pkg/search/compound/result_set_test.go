package compound

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestResultSet(t *testing.T) {
	suite.Run(t, new(ResultSetTestSuite))
}

type ResultSetTestSuite struct {
	suite.Suite
}

func (suite *ResultSetTestSuite) TestUnion() {
	results1 := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
		},
		{
			ID: "5",
		},
	}

	results2 := []search.Result{
		{
			ID: "5",
		},
		{
			ID: "3",
		},
		{
			ID: "6",
		},
	}

	expected := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
		},
		{
			ID: "5",
		},
		{
			ID: "6",
		},
	}

	rs1 := newResultSet(results1, false)
	rs2 := newResultSet(results2, false)
	actual := rs1.union(rs2)

	suite.Equal(expected, actual.asResultSlice())
}

func (suite *ResultSetTestSuite) TestIntersect() {
	results1 := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
		},
		{
			ID: "5",
		},
	}

	results2 := []search.Result{
		{
			ID: "3",
		},
		{
			ID: "5",
		},
		{
			ID: "6",
		},
	}

	expected := []search.Result{
		{
			ID: "3",
		},
		{
			ID: "5",
		},
	}

	rs1 := newResultSet(results1, false)
	rs2 := newResultSet(results2, false)
	actual := rs1.intersect(rs2)

	suite.Equal(expected, actual.asResultSlice())
}

func (suite *ResultSetTestSuite) TestSubtract() {
	results1 := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
		},
		{
			ID: "5",
		},
	}

	results2 := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "5",
		},
		{
			ID: "6",
		},
	}

	expected := []search.Result{
		{
			ID: "2",
		},
		{
			ID: "3",
		},
	}

	rs1 := newResultSet(results1, false)
	rs2 := newResultSet(results2, false)
	actual := rs1.subtract(rs2)

	suite.Equal(expected, actual.asResultSlice())
}

func (suite *ResultSetTestSuite) TestLeftJoin() {
	results1 := []search.Result{
		{
			ID: "1",
			Matches: map[string][]string{
				"1": {"a", "b"},
			},
		},
		{
			ID: "2",
			Matches: map[string][]string{
				"2": {"a", "b"},
			},
		},
		{
			ID: "3",
			Matches: map[string][]string{
				"3": {"a", "b"},
			},
		},
		{
			ID: "5",
			Matches: map[string][]string{
				"5": {"a", "b"},
			},
		},
	}

	results2 := []search.Result{
		{
			ID: "5",
		},
		{
			ID: "1",
		},
		{
			ID: "6",
		},
	}

	expected := []search.Result{
		{
			ID: "5",
			Matches: map[string][]string{
				"5": {"a", "b"},
			},
		},
		{
			ID: "1",
			Matches: map[string][]string{
				"1": {"a", "b"},
			},
		},
		{
			ID: "2",
			Matches: map[string][]string{
				"2": {"a", "b"},
			},
		},
		{
			ID: "3",
			Matches: map[string][]string{
				"3": {"a", "b"},
			},
		},
	}

	rs1 := newResultSet(results1, false)
	rs2 := newResultSet(results2, true)
	actual := rs1.leftJoinWithRightOrder(rs2)

	suite.Equal(expected, actual.asResultSlice())
}

func (suite *ResultSetTestSuite) TestMerge() {
	results1 := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
			Fields: map[string]interface{}{
				"f": "f1",
			},
			Matches: map[string][]string{},
		},
		{
			ID: "5",
			Fields: map[string]interface{}{
				"f": "f2",
			},
			Matches: map[string][]string{
				"m": {
					"m1",
					"m2",
				},
			},
		},
	}

	results2 := []search.Result{
		{
			ID: "3",
			Fields: map[string]interface{}{
				"g": "g1",
			},
			Matches: map[string][]string{
				"n": {
					"n1",
					"n2",
				},
			},
		},
		{
			ID: "5",
			Fields: map[string]interface{}{
				"f": "f2",
			},
			Matches: map[string][]string{
				"l": {
					"l1",
					"l2",
				},
			},
		},
		{
			ID: "6",
		},
	}

	expected := []search.Result{
		{
			ID: "3",
			Fields: map[string]interface{}{
				"f": "f1",
				"g": "g1",
			},
			Matches: map[string][]string{
				"n": {
					"n1",
					"n2",
				},
			},
		},
		{
			ID: "5",
			Fields: map[string]interface{}{
				"f": "f2",
			},
			Matches: map[string][]string{
				"l": {
					"l1",
					"l2",
				},
				"m": {
					"m1",
					"m2",
				},
			},
		},
	}

	rs1 := newResultSet(results1, false)
	rs2 := newResultSet(results2, false)
	actual := rs1.intersect(rs2)

	suite.Equal(expected, actual.asResultSlice())
}
