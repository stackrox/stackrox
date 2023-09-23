package jmespathcompile

import (
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/kyverno/go-jmespath"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	var jsondata = []byte(`{"foo": {"bar": {"baz": [0, 1, 2, 3, 4]}}}`) // your data
	var data interface{}
	err := json.Unmarshal(jsondata, &data)
	assert.NoError(t, err)
	compiled, err := jmespath.Compile("foo.bar.baz[2]")
	assert.NoError(t, err)
	result, err := compiled.Search(data)
	assert.Equal(t, result, int64(2))
}

type testCase struct {
	desc           string
	q              *query.Query
	obj            *TopLevel
	expectedResult *evaluator.Result
}

func assertResultsAsExpected(t *testing.T, c testCase, actualRes *evaluator.Result, actualMatched bool) {
	assert.Equal(t, c.expectedResult != nil, actualMatched)
	if c.expectedResult != nil {
		require.NotNil(t, actualRes)
		assert.ElementsMatch(t, c.expectedResult.Matches, actualRes.Matches)
	}
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

func TestBasicDeployment(t *testing.T) {
	// [{match:starts_[@][?ValA == `happy`].{ValA: ValA}with(ValA, `happy`), values: ValA}][?match].{TopLevelA: values}

	// [@][?starts_with(ValA, `happy`)].{ValA: ValA}
	qTopLevelAHappy := query.SimpleMatchFieldQuery("TopLevelA", "happy")
	// NestedSlice[?starts_with(NestedValA, `happy`)].{A: NestedValA}
	// [@][0].NestedSlice[?starts_with(NestedValA, `happy`)].{A: NestedValA}
	// qNestedAHappy := query.SimpleMatchFieldQuery("A", "happy")
	// NestedSlice[].SecondNestedSlice[?starts_with(SecondNestedValA, `happy`)].{A: SecondNestedValA}
	// qSecondNestedAHappy := query.SimpleMatchFieldQuery("SecondA", "r/.*ppy")
	// [{  "A": NestedSlice[?NestedValA == 'A1' && NestedValB == 'B1'].NestedValA,"B": NestedSlice[?NestedValA == 'A1' && NestedValB == 'B1'].NestedValB, "TopLevelA": [ValA] }] [?A != '[]' && B != `[]`]
	testCases := []testCase{
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
	}
	runTestCases(t, testCases)
}
func resultWithSingleMatch(fieldName string, values ...string) *evaluator.Result {
	return &evaluator.Result{Matches: []map[string][]string{{fieldName: values}}}
}

func runTestCases(t *testing.T, testCases []testCase) {
	for _, tk := range testCases {
		objStr, err := json.Marshal(tk.obj)
		assert.NoError(t, err)
		t.Log(string(objStr))
		compiled, err := jmespath.Compile("valA == 'happy'")
		assert.NoError(t, err)
		result, err := compiled.Search(tk.obj)
		assert.Equal(t, result, tk.obj.ValA == "happy")
	}
	for _, c := range testCases {
		objStr, err := json.Marshal(c.obj)
		assert.NoError(t, err)
		t.Log(string(objStr))
		compiled, err := jmespath.Compile("valA == 'happy'")
		assert.NoError(t, err)
		result, err := compiled.Search(c.obj)
		assert.Equal(t, result, c.obj.ValA == "happy")
		t.Run(c.desc, func(t *testing.T) {
			t.Run("on fully hydrated object", func(t *testing.T) {
				evaluator, err := compilerInstance.CompileJMESPathBasedEvaluator(c.q)
				require.NoError(t, err)
				res, matched := evaluator.Evaluate(pathutil.NewAugmentedObj(c.obj))
				assertResultsAsExpected(t, c, res, matched)
			})
			/*
				t.Run("on augmented object", func(t *testing.T) {
					evaluator, err := augmentedCompilerInstance.CompileRegoBasedEvaluator(c.q)
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
			*/
		})
	}
}
