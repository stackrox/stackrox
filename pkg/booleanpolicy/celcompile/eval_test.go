package celcompile

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stretchr/testify/assert"
)

var (
	xyz = testCase{
		desc: "linked, multilevel, should match (group test)",
		obj: &TopLevel{
			ValA: "TopLevelValA",
			NestedSlice: []Nested{
				{NestedValA: "A0", NestedValB: "B0"},
				{NestedValA: "A0", NestedValB: "B1"},
				{NestedValA: "A1", NestedValB: "B0"},
				{NestedValA: "A1", NestedValB: "B1"},
				{NestedValA: "A2", NestedValB: "B2"},
			},
		},
		q: &query.Query{
			FieldQueries: []*query.FieldQuery{
				{Field: "TopLevelA", Values: []string{"TopLevelValA"}},
				{Field: "A", Values: []string{"A1", "A2"}, Operator: query.Or},
				{Field: "B", Values: []string{"B1", "B2"}, Operator: query.Or},
				{Field: "SecondA", Values: []string{"happy"}},
			},
		},
		expectedResult: &evaluator.Result{
			Matches: []map[string][]string{
				{"TopLevelA": {"TopLevelValA"}, "A": {"A1"}, "B": {"B1"}},
				{"TopLevelA": {"TopLevelValA"}, "A": {"A2"}, "B": {"B2"}},
			},
		},
	}
)

func TestBasic(t *testing.T) {
	var jsondata = []byte(`
	 {"foo": {"bar": {"baz": [0, 1, 2, 3, 4]}}}`) // your data
	var data map[string]any
	err := json.Unmarshal(jsondata, &data)
	assert.NoError(t, err)
	prog, err := compile("obj.foo.with({\"x\":\"y\"})")
	assert.NoError(t, err)
	input := map[string]interface{}{"obj": data}
	input2 := map[string]interface{}{"obj": interface{}(xyz.obj)}
	out1, err := evaluate(prog, input)
	fmt.Print(out1)
	out2, err := evaluate(prog, input2)
	fmt.Print(out2)
}

var tplate = `
[obj]
   .map(n, obj.ValA.startsWith("TopLevelValA"), [{"TopLevelValA": [n.ValA]}])
   .map(n, obj.NestedSlice.map(k, k.NestedValA.startsWith("A1") || k.NestedValA.startsWith("A2"), n.map(t, t.with({"A": [k.NestedValA]}))).flatten())
   .map(n, obj.NestedSlice.map(k, k.NestedValB.startsWith("B1") || k.NestedValB.startsWith("B2"), n.map(t, t.with({"B": [k.NestedValB]}))).flatten())
`

var tplate1 = `
[obj]
   // .map(n, obj.ValA.startsWith("TopLevelValA"), [{"TopLevelValA": [n.ValA]}])
   // .map(n, [{"match": ["true"]}])
   // .map(n, [{"match": []}])
   .map(n, [{}])
   .map(n, obj.NestedSlice.map(k, k.NestedValA.startsWith("A1") || k.NestedValA.startsWith("A2"), n.map(t, t.with({"A": [k.NestedValA]}))).flatten())
   .map(n, obj.NestedSlice.map(k, k.NestedValB.startsWith("B1") || k.NestedValB.startsWith("B2"), n.map(t, t.with({"B": [k.NestedValB]}))).flatten())
   .map(n, obj.ValA.startsWith("TopLevelValA"), n.map(t, t.with({"TopLevelValA": [obj.ValA]})))
    // [{"TopLevelValA": [n.ValA]}])
`

var tplatex = `
[[{}]]
   .map(result, obj.ValA.startsWith("ToLevelValA"), result.map(t, t.with({"TopLevelValA": [obj.ValA]})))
   .map(result, obj.NestedSlice.map(k, k.NestedValA.startsWith("A1") || k.NestedValA.startsWith("A2"), result.map(entry, entry.with({"A": [k.NestedValA]}))).flatten())
   .map(result, obj.NestedSlice.map(k, k.NestedValB.startsWith("B1") || k.NestedValB.startsWith("B2"), result.map(entry, entry.with({"B": [k.NestedValB]}))).flatten())
`

var tplatexx = `
[[{}]]
   .map(result, obj.NestedSlice.map(k, k.NestedValB.startsWith("B1") || k.NestedValB.startsWith("B2"), result.map(entry, entry.with({"B": [k.NestedValB]}))).flatten())
   .filter(result, result.size() != 0)
   .map(result, obj.NestedSlice.map(k, k.NestedValA.startsWith("A1") || k.NestedValA.startsWith("A2"), result.map(entry, entry.with({"A": [k.NestedValA]}))).flatten())
   .filter(result, result.size() != 0)
   .map(result, obj.ValA.startsWith("TopLevelValA"), result.map(t, t.with({"TopLevelValA": [obj.ValA]})))
`

var tplatetxt = `
[[{}]]
   .map(
     result,
     obj.NestedSlice.map(
       k,
       (k.NestedValB.startsWith("B1") || k.NestedValB.startsWith("B2")) && (k.NestedValA.startsWith("A1") || k.NestedValA.startsWith("A2")),
       result.map(entry, entry.with({"B": [k.NestedValB], "A": [k.NestedValA]}))).flatten()
    )
   .filter(result, result.size() != 0)
   .map(result, obj.ValA.startsWith("TopLevelValA"), result.map(t, t.with({"TopLevelValA": [obj.ValA]})))
`

var tplate2 = `
[] +
[[{}]]
   .map(result, obj.ValA.startsWith("TopLevelValA"), result.map(t, t.with({"TopLevelValA": [obj.ValA]})))
   .map(
      result,
      obj.NestedSlice
        .filter(
          k,
          k.NestedValB.startsWith("B1") || k.NestedValB.startsWith("B2")
        )
        .filter(
          k,
          k.NestedValA.startsWith("A1") || k.NestedValA.startsWith("A2")
        )
        .map(
          k,
          result.map(entry, entry.with({"B": [k.NestedValB], "A": [k.NestedValA]}))
        ).flatten()
   )
   .filter(result, result.size() != 0)
   .flatten()
`

func TestBasicXX(t *testing.T) {
	prog, err := compile(tplate2)
	assert.NoError(t, err)
	jsonStr, err := json.Marshal(xyz.obj)
	assert.NoError(t, err)

	var data interface{}
	assert.NoError(t, json.Unmarshal(jsonStr, &data))
	input := map[string]interface{}{"obj": data}
	out, err := evaluate(prog, input)
	assert.NoError(t, err)
	fmt.Print(out)
}

func TestBasicYY(t *testing.T) {
	var tplat = `
{"a":1, "b":2}.with({"c":2, "d":4})
`
	prog, err := compile(tplat)
	assert.NoError(t, err)
	jsonStr, err := json.Marshal(xyz.obj)
	assert.NoError(t, err)

	var data interface{}
	assert.NoError(t, json.Unmarshal(jsonStr, &data))
	input := map[string]interface{}{"obj": data}
	out, err := evaluate(prog, input)
	assert.NoError(t, err)
	fmt.Print(out)
}
