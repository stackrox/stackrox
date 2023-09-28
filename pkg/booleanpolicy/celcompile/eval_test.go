package celcompile

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stretchr/testify/assert"
)

var (
	xyz = testCase{
		desc: "linked, multilevel, should match (group test)",
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

var (
	tplate44 = `
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
)

func TestBasicXX(t *testing.T) {
	tmpx := `
        []
        +[[{}]]
        		   .map(
        		      prevResults,
                      obj.NestedSlice
        		        .map(
        		          k,
        		          [[{}]]
        		   .map(
        		      prevResults,
                      k.SecondNestedSlice
        		        .map(
        		          k,
        		          [[{}]]
          .map(rs, k.SecondNestedValA.matches('^(?i:.*ppy)$'), rs.map(r, r.with({"SecondA": [k.SecondNestedValA]})))
        		           .map(rs, prevResults.map(p, rs.map(r, p.with(r))))
        		        )
        		   )
        		   .filter(r, r.size() != 0)
        		   .flatten()
        		           .map(rs, prevResults.map(p, rs.map(r, p.with(r))))
        		        )
        		   )
        		   .filter(r, r.size() != 0)
        		   .flatten()
        .flatten()
`
	tmpx = `
        []
        +[[{}]]
        	.map(
        	   prevResults,
               obj.NestedSlice
        	   .map(
        		   k,
                   [[{}]]
        		   .map(
        		      prevResults,
                      k.SecondNestedSlice
        		      .map(
        		         k,
        		         [[{}]]
                         .map(rs, k.SecondNestedValA.matches('^(?i:.*ppy)$'), rs.map(r, r.with({"SecondA": [k.SecondNestedValA]})))
                         .map(rs, prevResults.map(p, rs.map(r, p.with(r))))
                         .flatten()
        		         .filter(r, r.size() != 0)
        		        )
        		         .filter(r, r.size() != 0)
                        
        		   )
        		   .filter(r, r.size() != 0)
        		   .map(rs, prevResults.map(p, rs.flatten().map(r, p.with(r))))
        		   //.map(rs, [1, 2, rs])
        	   )
            )
           .filter(r, r.size() != 0)
           .flatten()
           //.flatten()
`
	tmpx = `
        []
        +[[{}]]
        		   .map(
        		      prevResults,
                      obj.NestedSlice
        		        .map(
        		          k,
        		          [[{}]]
        		   .map(
        		      prevResults,
                      k.SecondNestedSlice
        		        .map(
        		          k,
        		          [[{}]]
          .map(rs, k.SecondNestedValA.matches('^(?i:.*ppy)$'), [rs].flatten().map(r, r.with({"SecondA": [k.SecondNestedValA]})))
        		           .filter(r, [r].flatten().size() != 0)
        		           .map(rs, [prevResults].flatten().map(p, [rs].flatten().map(r, p.with(r))))
                            .flatten()
                		         .filter(r, [r].flatten().size() != 0)
        		        )
        		   )
        		   .filter(r, [r].flatten().size() != 0)
        		           .filter(r, [r].flatten().size() != 0)
        		           .map(rs, [prevResults].flatten().map(p, [rs].flatten().map(r, p.with(r))))
                            .flatten()
                		         .filter(r, [r].flatten().size() != 0)
        		        )
        		   )
        		   .filter(r, [r].flatten().size() != 0)
        .flatten()
`

	obj := &TopLevel{
		ValA: "happy",
		NestedSlice: []Nested{
			{NestedValA: "happy"},
			{NestedValA: "something else", SecondNestedSlice: []*SecondNested{
				{SecondNestedValA: "happiest"},
			}},
		},
	}
	prog, err := compile(tmpx)
	assert.NoError(t, err)
	jsonStr, err := json.Marshal(obj)
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

type test struct {
	a, b int
}

func (t test) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	panic("not required")
}

func (t test) ConvertToType(typeVal ref.Type) ref.Val {
	panic("not required")
}

func (t test) Equal(other ref.Val) ref.Val {

	o, ok := other.Value().(test)
	if ok {
		if o == t {
			return types.Bool(true)
		} else {
			return types.Bool(false)
		}
	} else {
		return types.ValOrErr(other, "%v is not of type Test", other)
	}
}

func (t test) Type() ref.Type {
	return TestType
}

func (t test) Receive(function string, overload string, args []ref.Val) ref.Val {

	return types.ValOrErr(TestType, "no such function - %s", function)
}

func TestBasicXE(t *testing.T) {
	var tplat = `
obj
`
	prog, err := compile(tplat)
	assert.NoError(t, err)
	jsonStr, err := json.Marshal(xyz.obj)
	assert.NoError(t, err)

	var data interface{}
	assert.NoError(t, json.Unmarshal(jsonStr, &data))
	in, err := interpreter.NewActivation(map[string]interface{}{"obj": storage.TestChild1{Id: "xx"}})
	assert.NoError(t, err)
	out, _, err := prog.Eval(in)
	assert.NoError(t, err)
	fmt.Print(out)
}
