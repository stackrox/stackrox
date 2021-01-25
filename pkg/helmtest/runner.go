package helmtest

import (
	"encoding/json"
	"path"
	"strings"
	"testing"

	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/runtime"
)

type runner struct {
	t    *testing.T
	test *Test
	tgt  *Target
}

func (r *runner) Assert() *assert.Assertions {
	return assert.New(r.t)
}

func (r *runner) Require() *require.Assertions {
	return require.New(r.t)
}

func (r *runner) instantiateWorld(renderVals chartutil.Values) map[string]interface{} {
	world := make(map[string]interface{})

	renderValsBytes, err := json.Marshal(renderVals)
	if err != nil {
		panic(errors.Wrap(err, "marshaling Helm render values to JSON"))
	}
	var helmRenderVals map[string]interface{}
	if err := json.Unmarshal(renderValsBytes, &helmRenderVals); err != nil {
		panic(errors.Wrap(err, "unmarshaling Helm render values"))
	}
	world["helm"] = helmRenderVals

	renderedTemplates, err := (&engine.Engine{}).Render(r.tgt.Chart, renderVals)

	if *r.test.ExpectError {
		r.Require().Error(err, "expected rendering to fail")
		world["error"] = err.Error()
		return world
	}

	r.Require().NoError(err, "expected rendering to succeed")

	var allObjects []interface{}

	for fileName, renderedContents := range renderedTemplates {
		// TODO: Support subcharts (even though we don't use them)
		if !r.Assert().Truef(stringutils.ConsumePrefix(&fileName, r.tgt.Chart.Name()+"/"), "unexpected file %s", fileName) {
			continue
		}

		if fileName == "templates/NOTES.txt" {
			world["notes"] = renderedContents
			continue
		}

		if !r.Assert().Equalf(".yaml", path.Ext(fileName), "unexpected file type for file %s", fileName) {
			continue
		}

		objs, err := k8sutil.UnstructuredFromYAMLMulti(renderedContents)
		if !r.Assert().NoErrorf(err, "parsing objects from file %s", fileName) {
			continue
		}

		for _, obj := range objs {
			kindPlural := strings.TrimSuffix(strings.ToLower(obj.GetKind()), "s") + "s"
			objsByKind, _ := world[kindPlural].(map[string]interface{})
			if objsByKind == nil {
				objsByKind = make(map[string]interface{})
				world[kindPlural] = objsByKind
			}
			if !r.Assert().NotContainsf(objsByKind, obj.GetName(), "duplicate object %s/%s", kindPlural, obj.GetName()) {
				continue
			}
			objsByKind[obj.GetName()] = obj.Object
			allObjects = append(allObjects, obj.Object)
		}
	}

	world["objects"] = allObjects
	return world
}

func (r *runner) Run() {
	var values map[string]interface{}
	releaseOpts := r.tgt.ReleaseOptions

	r.test.forEachScopeBottomUp(func(t *Test) {
		scopeVals := t.Values
		if len(scopeVals) == 0 {
			return
		}
		values = chartutil.CoalesceTables(values, runtime.DeepCopyJSON(scopeVals))
	})
	r.test.forEachScopeTopDown(func(t *Test) {
		rel := t.Release
		if rel == nil {
			return
		}
		rel.apply(&releaseOpts)
	})

	renderVals, err := chartutil.ToRenderValues(r.tgt.Chart, values, releaseOpts, r.tgt.Capabilities)
	r.Require().NoError(err, "failed to obtain render values")

	world := r.instantiateWorld(renderVals)
	r.evaluatePredicates(world)
}

func (r *runner) evaluatePredicates(world map[string]interface{}) {
	var allFuncDefs []*gojq.FuncDef
	var allPreds []*gojq.Query

	r.test.forEachScopeTopDown(func(t *Test) {
		allFuncDefs = append(allFuncDefs, t.funcDefs...)
		for _, pred := range t.predicates {
			predWithFuncs := *pred
			predWithFuncs.FuncDefs = make([]*gojq.FuncDef, 0, len(allFuncDefs)+len(pred.FuncDefs))
			predWithFuncs.FuncDefs = append(predWithFuncs.FuncDefs, allFuncDefs...)
			predWithFuncs.FuncDefs = append(predWithFuncs.FuncDefs, pred.FuncDefs...)
			allPreds = append(allPreds, &predWithFuncs)
		}
	})

	for _, pred := range allPreds {
		code, err := gojqCompile(pred)

		if !r.Assert().NoErrorf(err, "failed to compile predicate %q", pred) {
			continue
		}

		iter := code.Run(runtime.DeepCopyJSON(world))
		for result, ok := iter.Next(); ok; result, ok = iter.Next() {
			err, _ := result.(error)
			if errors.Is(err, errAssumptionViolation) {
				continue
			}
			if !r.Assert().NoErrorf(err, "failed to evaluate pred %q", pred) {
				continue
			}
			r.Assert().True(truthiness(result), "predicate %q evaluated to falsy result %v", pred, result)
		}
	}
}
