package evaluator

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TopLevel struct {
	ValA        string `search:"TopLevelA"`
	NestedSlice []Nested
	Base        Base
}

type Base struct {

	// These exist for testing base types.
	ValBaseSlice     []string          `search:"BaseSlice"`
	ValBasePtr       *string           `search:"BasePtr"`
	ValBaseBool      bool              `search:"BaseBool"`
	ValBaseTS        *types.Timestamp  `search:"BaseTS"`
	ValBaseInt       int               `search:"BaseInt"`
	ValBaseUint      uint              `search:"BaseUint"`
	ValBaseFloat     float64           `search:"BaseFloat"`
	ValBaseEnum      storage.Access    `search:"BaseEnum"`
	ValBaseMap       map[string]string `search:"BaseMap"`
	ValBaseStructPtr *random           `search:"BaseStructPtr"`
}

type random struct {
	Val string
}

type Nested struct {
	NestedValA        string `search:"A"`
	NestedValB        string `search:"B"`
	SecondNestedSlice []*SecondNested
}

type SecondNested struct {
	SecondNestedValA string `search:"SecondA"`
	SecondNestedValB string `search:"SecondB"`
}

// Bare versions of top level and nested for the sake of testing augmentation.
// See augmentedFactoryInstance for how these objects are augmented to appear
// like they are their non-bare versions.
type TopLevelBare struct {
	ValA string `search:"TopLevelA"`
	// This struct will be replaced with the other "Base" by the augmentation.
	// We keep it here to make sure the augmentation correctly supplants this object.
	Base struct {
		IrrelevantBaseVal int `search:"BaseInt"`
	}
}

type NestedBare struct {
	NestedValA string `search:"A"`
	NestedValB string `search:"B"`
}

var (
	factoryInstance = MustCreateNewFactory(pathutil.NewAugmentedObjMeta((*TopLevel)(nil)))

	augmentedFactoryInstance = MustCreateNewFactory(pathutil.NewAugmentedObjMeta((*TopLevelBare)(nil)).
					AddPlainObjectAt([]string{
			"Base"}, Base{}).
		AddAugmentedObjectAt(
			[]string{
				"NestedSlice"},
			pathutil.NewAugmentedObjMeta(([]NestedBare)(nil)).
				AddPlainObjectAt([]string{"SecondNestedSlice"}, ([]*SecondNested)(nil)),
		),
	)

	ts2020Apr01 = protoconv.MustConvertTimeToTimestamp(timeutil.MustParse(time.RFC3339, "2020-04-01T00:00:00Z"))
)

type testCase struct {
	desc           string
	q              *query.Query
	obj            *TopLevel
	expectedResult *Result
}

func assertResultsAsExpected(t *testing.T, c testCase, actualRes *Result, actualMatched bool) {
	assert.Equal(t, c.expectedResult != nil, actualMatched)
	if c.expectedResult != nil {
		require.NotNil(t, actualRes)
		assert.ElementsMatch(t, c.expectedResult.Matches, actualRes.Matches)
	}
}

func resultWithSingleMatch(fieldName string, values ...string) *Result {
	return &Result{[]map[string][]string{{fieldName: values}}}
}

func (r *Result) addContextMatch(fieldName string, values ...string) *Result {
	for _, match := range r.Matches {
		match[fieldName] = values
	}
	r.Matches = append(r.Matches, map[string][]string{fieldName: values})
	return r
}

func runTestCases(t *testing.T, testCases []testCase) {
	for _, testCase := range testCases {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			t.Run("on fully hydrated object", func(t *testing.T) {
				evaluator, err := factoryInstance.GenerateEvaluator(c.q)
				require.NoError(t, err)
				res, matched := evaluator.Evaluate(pathutil.NewAugmentedObj(c.obj))
				assertResultsAsExpected(t, c, res, matched)
			})
			t.Run("on augmented object", func(t *testing.T) {
				evaluator, err := augmentedFactoryInstance.GenerateEvaluator(c.q)
				require.NoError(t, err)
				topLevelBare := &TopLevelBare{
					ValA: c.obj.ValA,
				}
				base := c.obj.Base
				nestedBare := make([]NestedBare, 0, len(c.obj.NestedSlice))
				for _, elem := range c.obj.NestedSlice {
					nestedBare = append(nestedBare, NestedBare{NestedValA: elem.NestedValA, NestedValB: elem.NestedValB})
				}

				nestedAugmentedObj := pathutil.NewAugmentedObj(nestedBare)
				for i, elem := range c.obj.NestedSlice {
					require.NoError(t, nestedAugmentedObj.AddPlainObjAt(elem.SecondNestedSlice, pathutil.IndexStep(i), pathutil.FieldStep("SecondNestedSlice")))
				}
				topLevelAugmentedObj := pathutil.NewAugmentedObj(topLevelBare)
				require.NoError(t, topLevelAugmentedObj.AddPlainObjAt(base, pathutil.FieldStep("Base")))
				require.NoError(t, topLevelAugmentedObj.AddAugmentedObjAt(nestedAugmentedObj, pathutil.FieldStep("NestedSlice")))
				res, matched := evaluator.Evaluate(topLevelAugmentedObj)
				assertResultsAsExpected(t, c, res, matched)
			})
		})
	}
}

func TestMap(t *testing.T) {
	containsX3 := query.MapShouldContain("x", "3")

	containsValLucky := query.MapShouldContain("", "lucky")
	notContainsKeyHappy := query.MapShouldNotContain("happy", "")
	notContainsValHappy := query.MapShouldNotContain("", "happy")
	notContainsValLucky := query.MapShouldNotContain("", "lucky")

	runTestCases(t, []testCase{
		{
			desc: "simple map, required query, doesnt match",
			q:    query.SimpleMatchFieldQuery("BaseMap", notContainsValHappy),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"x": "happy",
					}},
			},
		},
		{
			desc: "simple map, required query, matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", notContainsValHappy),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"x": "y",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "x=y"),
		},

		{
			desc: "simple map, disallowed query, doesnt match",
			q:    query.SimpleMatchFieldQuery("BaseMap", containsX3),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"x": "y",
					}},
			},
		},

		{
			desc: "simple map, disallowed query, matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", containsX3),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"b": "z",
						"a": "y",
						"x": "3",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "x=3"),
		},

		{
			desc: "simple map, disallowed & required query, matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(containsX3, notContainsValHappy)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"b": "z",
						"a": "y",
						"x": "3",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "x=3"),
		},

		{
			desc: "simple map, disallowed & required query, matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(containsX3, notContainsValHappy)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "<empty>"),
		},

		{
			desc: "simple map, disallowed & required query, does not match",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(containsX3, notContainsValHappy)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"a": "happy",
					}},
			},
		},

		{
			desc: "simple map, disallowed & required query, matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(containsX3, notContainsValHappy)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"happy": "a",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "happy=a"),
		},

		{
			desc: "simple map, disallowed & required query 2, matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(containsX3, notContainsKeyHappy)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"b": "z",
						"a": "y",
						"x": "3",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "x=3"),
		},

		{
			desc: "simple map, disallowed & required query 2, matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(containsX3, notContainsKeyHappy)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "<empty>"),
		},

		{
			desc: "simple map, disallowed & required query 2, does not match",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(containsX3, notContainsKeyHappy)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"happy": "a",
					}},
			},
		},

		{
			desc: "simple map, disallowed & required query 2, matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(containsX3, notContainsKeyHappy)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"a": "happy",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "a=happy"),
		},

		{
			desc: "simple map, required disjunction query , matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(notContainsKeyHappy, notContainsValLucky)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"a": "happy",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "a=happy"),
		},

		{
			desc: "simple map, required disjunction query , matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(notContainsKeyHappy, notContainsValLucky)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"a": "lucky",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "a=lucky"),
		},

		{
			desc: "simple map, required disjunction query , does not match",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(notContainsKeyHappy, notContainsValLucky)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"happy": "lucky",
					}},
			},
		},

		{
			desc: "simple map, required conjunction query , does not match",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAllOf(notContainsKeyHappy, notContainsValLucky)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"happy": "lucky",
					}},
			},
		},

		{
			desc: "simple map, required conjunction query , does not match",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAllOf(notContainsKeyHappy, notContainsValLucky)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"a": "lucky",
					}},
			},
		},

		{
			desc: "simple map, required conjunction query , matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAllOf(notContainsKeyHappy, notContainsValLucky)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"lucky": "happy",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "lucky=happy"),
		},

		{
			desc: "simple map, required disallowed conjunction query , does not match",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAllOf(notContainsKeyHappy, containsValLucky)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"happy": "lucky",
					}},
			},
		},

		{
			desc: "simple map, required disallowed conjunction query , matches",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAllOf(notContainsKeyHappy, containsValLucky)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"a": "lucky",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "a=lucky"),
		},

		{
			desc: "simple map, required disallowed conjunction query , does not match",
			q:    query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAllOf(notContainsKeyHappy, containsValLucky)),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"lucky": "happy",
					}},
			},
		},

		{
			desc: "simple map, complex query , does not match",
			q: query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(
				query.MapShouldMatchAllOf(notContainsKeyHappy, containsValLucky), query.MapShouldContain("happy", "true"))),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"happy": "lucky",
					}},
			},
		},

		{
			desc: "simple map, complex query , does not match",
			q: query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(
				query.MapShouldMatchAllOf(notContainsKeyHappy, containsValLucky), query.MapShouldContain("happy", "true"))),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"a":     "lucky",
						"happy": "1",
					}},
			},
		},

		{
			desc: "simple map, complex query , matches",
			q: query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(
				query.MapShouldMatchAllOf(notContainsKeyHappy, containsValLucky), query.MapShouldContain("happy", "true"))),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"a":     "lucky",
						"happy": "true",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "happy=true"),
		},

		{
			desc: "simple map, complex query , matches",
			q: query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(
				query.MapShouldMatchAllOf(notContainsKeyHappy, containsValLucky), query.MapShouldContain("happy", "true"))),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"lucky": "happy",
						"happy": "true",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "happy=true"),
		},

		{
			desc: "simple map, complex query last, matches",
			q: query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(
				query.MapShouldMatchAllOf(notContainsKeyHappy, containsValLucky), query.MapShouldContain("happy", "true"))),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"true": "lucky",
						"blah": "bleh",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "true=lucky"),
		},

		{
			desc: "simple map, complex query extra k,v pairs, matches",
			q: query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(
				query.MapShouldMatchAllOf(notContainsKeyHappy, notContainsValLucky), query.MapShouldContain("happy", "true"))),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"a":    "notluckyalsoveryveryveryveryveryveryvvveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong",
						"blah": "bleh",
						"k1":   "v1",
						"k2":   "v2",
						"k3":   "v3",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap",
				"a=notluckyalsoveryveryveryveryveryveryvvveryveryveryveryveryveryve, blah=bleh, k1=v1 and 2 more"),
		},

		{
			desc: "simple map, complex query extra k,v pairs(dont take extra), matches",
			q: query.SimpleMatchFieldQuery("BaseMap", query.MapShouldMatchAnyOf(
				query.MapShouldMatchAllOf(notContainsKeyHappy, containsValLucky), query.MapShouldContain("happy", "true"))),
			obj: &TopLevel{
				ValA: "whatever",
				Base: Base{
					ValBaseMap: map[string]string{
						"lucky": "happy",
						"happy": "true",
						"k1":    "v1",
						"k2":    "v2",
						"k3":    "v3",
						"k4":    "v4",
					}},
			},
			expectedResult: resultWithSingleMatch("BaseMap", "happy=true"),
		},
	})
}

func TestSimpleBase(t *testing.T) {
	qTopLevelAHappy := query.SimpleMatchFieldQuery("TopLevelA", "happy")
	qNestedAHappy := query.SimpleMatchFieldQuery("A", "happy")
	qSecondNestedAHappy := query.SimpleMatchFieldQuery("SecondA", "r/.*ppy")

	runTestCases(t, []testCase{
		{
			desc: "simple one for top level, doesn't pass",
			q:    qTopLevelAHappy,
			obj: &TopLevel{
				ValA: "whatever",
				NestedSlice: []Nested{
					{NestedValA: "blah"},
					{NestedValA: "something else", SecondNestedSlice: []*SecondNested{
						{SecondNestedValA: "happy"},
					}},
				},
			},
		},
		{
			desc: "simple one for top level, passes",
			q:    qTopLevelAHappy,
			obj: &TopLevel{
				ValA: "happy",
				NestedSlice: []Nested{
					{NestedValA: "blah"},
					{NestedValA: "something else", SecondNestedSlice: []*SecondNested{
						{SecondNestedValA: "happy"},
					}},
				},
			},
			expectedResult: resultWithSingleMatch("TopLevelA", "happy"),
		},
		{
			desc: "simple one for first level nested, doesn't pass",
			q:    qNestedAHappy,
			obj: &TopLevel{
				ValA: "happy",
				NestedSlice: []Nested{
					{NestedValA: "blah"},
					{NestedValA: "something else", SecondNestedSlice: []*SecondNested{
						{SecondNestedValA: "happy"},
					}},
				},
			},
		},
		{
			desc: "simple one for first level nested, passes",
			q:    qNestedAHappy,
			obj: &TopLevel{
				ValA: "happy",
				NestedSlice: []Nested{
					{NestedValA: "happy"},
					{NestedValA: "something else", SecondNestedSlice: []*SecondNested{
						{SecondNestedValA: "happiest"},
					}},
				},
			},
			expectedResult: resultWithSingleMatch("A", "happy"),
		},
		{
			desc: "simple one for second level nested, doesn't pass",
			q:    qSecondNestedAHappy,
			obj: &TopLevel{
				ValA: "happy",
				NestedSlice: []Nested{
					{NestedValA: "happy"},
					{NestedValA: "something else", SecondNestedSlice: []*SecondNested{
						{SecondNestedValA: "happiest"},
					}},
				},
			},
		},
		{
			desc: "simple one for second level nested, passes",
			q:    qSecondNestedAHappy,
			obj: &TopLevel{
				ValA: "happy",
				NestedSlice: []Nested{
					{NestedValA: "happy", SecondNestedSlice: []*SecondNested{
						{SecondNestedValA: "blah"},
						{SecondNestedValA: "blaappy"},
					}},
					{NestedValA: "something else", SecondNestedSlice: []*SecondNested{
						{SecondNestedValA: "happy"},
					}},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"SecondA": {"happy"}},
					{"SecondA": {"blaappy"}},
				},
			},
		},
	})
}

func TestLinked(t *testing.T) {
	runTestCases(t, []testCase{
		{
			desc: "linked, first level of nesting, should match",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{"A1"}},
					{Field: "B", Values: []string{"B1"}},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"A": {"A1"}, "B": {"B1"}},
				},
			},
		},
		{
			desc: "linked, first level of nesting, should not match",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{"A0"}},
					{Field: "B", Values: []string{"B1"}},
				},
			},
		},
		{
			desc: "linked, multilevel, should match",
			obj: &TopLevel{
				ValA: "TopLevelValA",
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "TopLevelA", Values: []string{"TopLevelValA"}},
					{Field: "A", Values: []string{"A1"}},
					{Field: "B", Values: []string{"B1"}},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"TopLevelA": {"TopLevelValA"}, "A": {"A1"}, "B": {"B1"}},
				},
			},
		},
		{
			desc: "linked, multilevel, should match (group test)",
			obj: &TopLevel{
				ValA: "TopLevelValA",
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
					{NestedValA: "A2", NestedValB: "B2"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "TopLevelA", Values: []string{"TopLevelValA"}},
					{Field: "A", Values: []string{"A1", "A2"}, Operator: query.Or},
					{Field: "B", Values: []string{"B1", "B2"}, Operator: query.Or},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"TopLevelA": {"TopLevelValA"}, "A": {"A1"}, "B": {"B1"}},
					{"TopLevelA": {"TopLevelValA"}, "A": {"A2"}, "B": {"B2"}},
				},
			},
		},
		{
			desc: "linked, multilevel, top doesn't match",
			obj: &TopLevel{
				ValA: "TopLevelValA",
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "TopLevelA", Values: []string{"NONEXISTENT"}},
					{Field: "A", Values: []string{"A1"}},
					{Field: "B", Values: []string{"B1"}},
				},
			},
		},
		{
			desc: "linked, multilevel, bottom doesn't match",
			obj: &TopLevel{
				ValA: "TopLevelValA",
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "TopLevelA", Values: []string{"TopLevelValA"}},
					{Field: "A", Values: []string{"A0"}},
					{Field: "B", Values: []string{"B1"}},
				},
			},
		},
	})
}

func TestSliceBase(t *testing.T) {
	runTestCases(t, []testCase{
		{
			desc: "slice base, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one"}},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"BaseSlice": {"one"}},
				},
			},
		},
		{
			desc: "slice base, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"four"}},
				},
			},
		},
		{
			desc: "slice base, with OR, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "four"}, Operator: query.Or},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"BaseSlice": {"one"}},
				},
			},
		},
		{
			desc: "slice base, with OR, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"five", "four"}, Operator: query.Or},
				},
			},
		},
		{
			desc: "slice base, with AND, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "four"}, Operator: query.And},
				},
			},
		},
		{
			desc: "slice base, with AND, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "two"}, Operator: query.And},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"BaseSlice": {"one", "two"}},
				},
			},
		},
		{
			desc: "empty slice, simple query",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "two"}, Operator: query.Or},
				},
			},
		},
		{
			desc: "empty slice, AND query",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "two"}, Operator: query.And},
				},
			},
		},

		{
			desc: "empty slice, negated query",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "two"}, Operator: query.Or, Negate: true},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"BaseSlice": {"<empty>"}},
				},
			},
		},
		{
			desc: "empty slice, negated AND query",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "two"}, Operator: query.And, Negate: true},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"BaseSlice": {"<empty>"}},
				},
			},
		},
		{
			desc: "slice base, simple negated, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one"}, Negate: true},
				},
			},
		},
		{
			desc: "slice base, simple negated, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"four"}, Negate: true},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"BaseSlice": {"one", "two", "three"}},
				},
			},
		},
		{
			desc: "slice base, with OR, negated, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "four"}, Operator: query.Or, Negate: true},
				},
			},
		},
		{
			desc: "slice base, with OR, negated, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"five", "four"}, Operator: query.Or, Negate: true},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"BaseSlice": {"one", "two", "three"}},
				},
			},
		},
		{
			desc: "slice base, negated, with AND, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "four"}, Operator: query.And, Negate: true},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"BaseSlice": {"two", "three", "one"}},
				},
			},
		},
		{
			desc: "slice base, negated, with AND, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseSlice: []string{"one", "two", "three"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseSlice", Values: []string{"one", "two"}, Operator: query.And, Negate: true},
				},
			},
		},
	})
}

func TestCompound(t *testing.T) {
	runTestCases(t, []testCase{
		{
			desc: "simple compound query, OR, matches",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{"A0", "A1"}, Operator: query.Or},
				},
			},
			expectedResult: &Result{
				Matches: []map[string][]string{
					{"A": {"A0"}},
					{"A": {"A1"}},
				},
			},
		},
		{
			desc: "simple compound query, OR, does not match",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{"A2", "A3"}, Operator: query.Or},
				},
			},
		},
		{
			desc: "simple compound query, AND, does not match",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{"A0", "A1"}, Operator: query.And},
				},
			},
		},
		{
			desc: "simple compound query, AND, matches",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{"r/A.*", "r/.*1"}, Operator: query.And},
				},
			},
			expectedResult: resultWithSingleMatch("A", "A1"),
		},
		{
			desc: "compound query, OR, negated, matches",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{"A2", "A1"}, Operator: query.Or, Negate: true},
				},
			},
			expectedResult: resultWithSingleMatch("A", "A0"),
		},
		{
			desc: "compound query, OR, negated, does not match",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{"A0", "A1"}, Operator: query.Or, Negate: true},
				},
			},
		},
		{
			desc: "compound query, AND, negated, does not match",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{`r/A.*`, `r/.*\d`}, Operator: query.And, Negate: true},
				},
			},
		},
		{
			desc: "simple compound query, AND, negated, matches",
			obj: &TopLevel{
				NestedSlice: []Nested{
					{NestedValA: "A0", NestedValB: "B0"},
					{NestedValA: "A1", NestedValB: "B1"},
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "A", Values: []string{"r/A.*", "r/.*1"}, Operator: query.And, Negate: true},
				},
			},
			expectedResult: resultWithSingleMatch("A", "A0"),
		},
	})
}

func TestDifferentBaseTypes(t *testing.T) {
	runTestCases(t, []testCase{
		{
			desc: "base ptr, null query, nil pointer",
			obj: &TopLevel{
				Base: Base{
					ValBasePtr: nil},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BasePtr", Values: []string{"-"}},
				},
			},
			expectedResult: resultWithSingleMatch("BasePtr", "<nil>"),
		},
		{
			desc: "base ptr, not null query, nil pointer",
			obj: &TopLevel{
				Base: Base{
					ValBasePtr: nil},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BasePtr", Values: []string{"-"}, Negate: true},
				},
			},
		},
		{
			desc: "base ptr, null query, non-nil",
			obj: &TopLevel{
				Base: Base{
					ValBasePtr: pointers.String("anything")},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BasePtr", Values: []string{"-"}},
				},
			},
		},
		{
			desc: "base ptr, not null query, non-nil",
			obj: &TopLevel{
				Base: Base{
					ValBasePtr: pointers.String("anything")},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BasePtr", Values: []string{"-"}, Negate: true},
				},
			},
			expectedResult: resultWithSingleMatch("BasePtr", "anything"),
		},
		{
			desc: "base ptr, regular string query, matches",
			obj: &TopLevel{
				Base: Base{
					ValBasePtr: pointers.String("happy")},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BasePtr", Values: []string{"happy"}},
				},
			},
			expectedResult: resultWithSingleMatch("BasePtr", "happy"),
		},
		{
			desc: "base ptr, regular string query, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBasePtr: pointers.String("nothappy")},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BasePtr", Values: []string{"happy"}},
				},
			},
		},
		{
			desc: "base bool, should match",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseBool", Values: []string{"false"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseBool", "false"),
		},
		{
			desc: "base bool, should not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseBool: true},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseBool", Values: []string{"false"}},
				},
			},
		},
		{
			// This is pretty pointless practically, but our code _should_
			// correctly handle it.
			desc: "base bool, with negation",
			obj: &TopLevel{
				Base: Base{
					ValBaseBool: true},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseBool", Values: []string{"false"}, Negate: true},
				},
			},
			expectedResult: resultWithSingleMatch("BaseBool", "true"),
		},
		{
			desc: "base ts, null, matches",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{"-"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseTS", "<empty timestamp>"),
		},
		{
			desc: "base ts, null query, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseTS: ts2020Apr01},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{"-"}},
				},
			},
		},
		{
			desc: "base ts, null ts, but valid query, does not match",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{"<05/01/2020"}},
				},
			},
		},
		{
			desc: "base ts, null ts, not null query, does not match",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{"-"}, Negate: true},
				},
			},
		},
		{
			desc: "base ts, null ts, but valid query, negated, does not match",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{"<05/01/2020"}, Negate: true},
				},
			},
		},
		{
			desc: "base ts, valid ts, not null query, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseTS: ts2020Apr01},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{"-"}, Negate: true},
				},
			},
			expectedResult: resultWithSingleMatch("BaseTS", "2020-04-01 00:00:00"),
		},
		{
			desc: "base ts, query by absolute, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseTS: ts2020Apr01},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{"<05/01/2020"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseTS", "2020-04-01 00:00:00"),
		},
		{
			desc: "base ts, query by absolute, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseTS: ts2020Apr01},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{">05/01/2020"}},
				},
			},
		},
		{
			desc: "base ts, query by absolute, negate",
			obj: &TopLevel{
				Base: Base{
					ValBaseTS: ts2020Apr01},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{">05/01/2020"}, Negate: true},
				},
			},
			expectedResult: resultWithSingleMatch("BaseTS", "2020-04-01 00:00:00"),
		},
		{
			desc: "base ts, query by relative, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseTS: ts2020Apr01},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseTS", Values: []string{">20d"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseTS", "2020-04-01 00:00:00"),
		},
		{
			desc: "base ts, query by relative, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseTS: ts2020Apr01},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					// This test will fail in 1000 years, but if this test still matters then,
					// we have other problems...
					{Field: "BaseTS", Values: []string{">01/01/3023"}},
				},
			},
		},
		{
			desc: "base int, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseInt: 1},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseInt", Values: []string{"<2"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseInt", "1"),
		},
		{
			desc: "base int, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseInt: 1,
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseInt", Values: []string{">2"}},
				},
			},
		},
		{
			desc: "base uint, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseUint: 1,
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseUint", Values: []string{"<2"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseUint", "1"),
		},
		{
			desc: "base uint, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseUint: 1,
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseUint", Values: []string{">2"}},
				},
			},
		},
		{
			desc: "base float, matches and is a whole number",
			obj: &TopLevel{
				Base: Base{
					ValBaseFloat: 1.0,
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseFloat", Values: []string{">0.99"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseFloat", "1"),
		},
		{
			desc: "base float, exact match",
			obj: &TopLevel{
				Base: Base{
					// This is not exactly 1.1, but close enough, so make sure that
					// we treat this as equal.
					ValBaseFloat: 35337666376.11 / 32125151251.,
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseFloat", Values: []string{"1.1"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseFloat", "1.1"),
		},
		{
			desc: "base float, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseFloat: 1.1,
				},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseFloat", Values: []string{"<1.11"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseFloat", "1.1"),
		},
		{
			desc: "base float, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseFloat: 1.1},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseFloat", Values: []string{">1.1"}},
				},
			},
		},
		{
			desc: "base enum, exact, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseEnum: storage.Access_READ_ACCESS},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseEnum", Values: []string{"READ_ACCESS"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseEnum", "read_access"),
		},
		{
			desc: "base enum, exact, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseEnum: storage.Access_READ_ACCESS},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseEnum", Values: []string{"READ_WRITE_ACCESS"}},
				},
			},
		},
		{
			desc: "base enum, range, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseEnum: storage.Access_READ_WRITE_ACCESS},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseEnum", Values: []string{">=READ_ACCESS"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseEnum", "read_write_access"),
		},
		{
			desc: "base enum, range, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseEnum: storage.Access_READ_ACCESS},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseEnum", Values: []string{"<READ_ACCESS"}},
				},
			},
		},
		{
			desc: "base enum, complex range, matches",
			obj: &TopLevel{
				Base: Base{
					ValBaseEnum: storage.Access_READ_ACCESS},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{Field: "BaseEnum", Values: []string{">NO_ACCESS", "<READ_WRITE_ACCESS"}, Operator: query.And},
				},
			},
			expectedResult: resultWithSingleMatch("BaseEnum", "read_access"),
		},
		{
			desc: "base enum, complex range, does not match",
			obj: &TopLevel{
				Base: Base{
					ValBaseEnum: storage.Access_READ_WRITE_ACCESS},
			},
			q: &query.Query{

				FieldQueries: []*query.FieldQuery{
					{
						Field: "BaseEnum", Values: []string{">NO_ACCESS", "<READ_WRITE_ACCESS"}, Operator: query.And},
				},
			},
		},
		{
			desc: "base struct ptr, nil, matches",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{
						Field: "BaseStructPtr", Values: []string{"-"}},
				},
			},
			expectedResult: resultWithSingleMatch("BaseStructPtr", "<nil>"),
		},
		{
			desc: "base struct ptr, not nil, does not match",
			obj: &TopLevel{
				Base: Base{ValBaseStructPtr: &random{"asa"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{
						Field: "BaseStructPtr", Values: []string{"-"}},
				},
			},
		},
		{
			desc: "base struct ptr, nil, negated, does not match",
			obj:  &TopLevel{},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{
						Field: "BaseStructPtr", Values: []string{"-"}, Negate: true},
				},
			},
		},
		{
			desc: "base struct ptr, not nil, negated, matches",
			obj: &TopLevel{
				Base: Base{ValBaseStructPtr: &random{"asa"}},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					{
						Field: "BaseStructPtr", Values: []string{"-"}, Negate: true},
				},
			},
			expectedResult: resultWithSingleMatch("BaseStructPtr", "<non-nil>"),
		},
	})
}

func TestDifferentBaseTypesMatchAll(t *testing.T) {
	nestedFieldQuery := &query.FieldQuery{Field: "A", Values: []string{"A0"}}
	nestedFieldQueryNegated := &query.FieldQuery{Field: "A", Values: []string{"A0"}, Negate: true}
	runTestCases(t, []testCase{
		{
			desc: "base ptr, nil pointer",
			obj: &TopLevel{
				NestedSlice: []Nested{{NestedValA: "A0"}},
				Base: Base{
					ValBasePtr: nil},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					nestedFieldQuery,
					{Field: "BasePtr", MatchAll: true},
				},
			},
			expectedResult: resultWithSingleMatch("A", "A0").addContextMatch("BasePtr", "<nil>"),
		},
		{
			desc: "base ptr, nil pointer, no match",
			obj: &TopLevel{
				NestedSlice: []Nested{{NestedValA: "nomatch"}},
				Base: Base{
					ValBasePtr: nil},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					nestedFieldQuery,
					{Field: "BasePtr", MatchAll: true},
				},
			},
		},
		{
			desc: "base ptr, nil pointer, negated",
			obj: &TopLevel{
				NestedSlice: []Nested{{NestedValA: "asdf"}},
				Base: Base{
					ValBasePtr: nil},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					nestedFieldQueryNegated,
					{Field: "BasePtr", MatchAll: true},
				},
			},
			expectedResult: resultWithSingleMatch("A", "asdf").addContextMatch("BasePtr", "<nil>"),
		},
		{
			desc: "base ptr, nil pointer, negated, no match",
			obj: &TopLevel{
				NestedSlice: []Nested{{NestedValA: "A0"}},
				Base: Base{
					ValBasePtr: nil},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					nestedFieldQueryNegated,
					{Field: "BasePtr", MatchAll: true},
				},
			},
		},
		{
			desc: "base ptr, not null query, match",
			obj: &TopLevel{
				NestedSlice: []Nested{{NestedValA: "A0"}},
				Base: Base{
					ValBasePtr: pointers.String("anything")},
			},
			q: &query.Query{
				FieldQueries: []*query.FieldQuery{
					nestedFieldQuery,
					{Field: "BasePtr", MatchAll: true},
				},
			},
			expectedResult: resultWithSingleMatch("A", "A0").addContextMatch("BasePtr", "anything"),
		},
		// TODO(rc) test all base types for MatchAll paths
	})
}
