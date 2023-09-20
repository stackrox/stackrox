package regocompile

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/utils"
)

type regoCompilerForType struct {
	fieldToMetaPathMap *pathutil.FieldToMetaPathMap
}

// A RegoCompiler compiles a rego-based evaluator for the given query.
type RegoCompiler interface {
	CompileRegoBasedEvaluator(query *query.Query) (evaluator.Evaluator, error)
}

type regoBasedEvaluator struct {
	q      rego.PreparedEvalQuery
	module string // For debug
}

// convertBindingToResult converts a set of variable bindings to a result.
// It has to do a bunch of type assertions, since rego can return arbitrary values.
// We know that our rego programs are constructed to return map[string][]interface{},
// so this takes advantage of that to traverse them. It also converts each returned value
// into a string.
func convertBindingToResult(binding interface{}) (m map[string][]string, err error) {
	panicked := true
	defer func() {
		if r := recover(); r != nil || panicked {
			err = fmt.Errorf("panic running evaluator: %v", r)
		}
	}()
	m = make(map[string][]string)
	for k, v := range binding.(map[string]interface{}) {
		vAsInterfaceSlice := v.([]interface{})
		vAsString := make([]string, 0, len(vAsInterfaceSlice))
		for _, val := range vAsInterfaceSlice {
			vAsString = append(vAsString, fmt.Sprintf("%v", val))
		}
		m[k] = vAsString
	}
	panicked = false
	return m, nil
}

func (r *regoBasedEvaluator) Evaluate(obj *pathutil.AugmentedObj) (*evaluator.Result, bool) {
	parsedInput, err := obj.ParsedRegoValue()
	// If there is an error here, it is a programming error. Let's not panic in prod over it.
	if err != nil {
		utils.Should(err)
		return nil, false
	}

	resultSet, err := r.q.Eval(context.Background(), rego.EvalParsedInput(parsedInput))
	// If there is an error here, it is a programming error. Let's not panic in prod over it.
	if err != nil {
		utils.Should(err)
		return nil, false
	}
	if len(resultSet) != 1 {
		utils.Should(fmt.Errorf("invalid resultSet: %+v", resultSet))
		return nil, false
	}
	result := resultSet[0]
	outBindings, found := result.Bindings["out"].([]interface{})
	if !found {
		utils.Should(errors.New("resultSet didn't contain the expected bindings"))
		return nil, false
	}

	// This means it didn't match.
	if len(outBindings) == 0 {
		return nil, false
	}

	res := &evaluator.Result{}
	for _, binding := range outBindings {
		match, err := convertBindingToResult(binding)
		if err != nil {
			utils.Should(fmt.Errorf("failed to convert binding %+v: %w", binding, err))
			return nil, false
		}
		res.Matches = append(res.Matches, match)
	}
	return res, true
}

// MustCreateRegoCompiler is a wrapper around CreateRegoCompiler that panics if there's an error.
func MustCreateRegoCompiler(objMeta *pathutil.AugmentedObjMeta) RegoCompiler {
	r, err := CreateRegoCompiler(objMeta)
	utils.Must(err)
	return r
}

// CreateRegoCompiler creates a rego compiler for the given object meta.
func CreateRegoCompiler(objMeta *pathutil.AugmentedObjMeta) (RegoCompiler, error) {
	fieldToMetaPathMap, err := objMeta.MapSearchTagsToPaths()
	if err != nil {
		return nil, err
	}
	return &regoCompilerForType{fieldToMetaPathMap: fieldToMetaPathMap}, nil
}

func (r *regoCompilerForType) CompileRegoBasedEvaluator(query *query.Query) (evaluator.Evaluator, error) {
	regoModule, err := r.compileRego(query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile rego: %w", err)
	}
	q, err := rego.New(
		rego.Query("out = data.policy.main.violations"),
		rego.Module("main.policy", regoModule),
	).PrepareForEval(context.Background())
	if err != nil {
		return nil, err
	}
	return &regoBasedEvaluator{q: q, module: regoModule}, nil
}

type fieldMatchData struct {
	matchers []regoMatchFunc
	name     string
	path     string
}

func (r *regoCompilerForType) compileRego(query *query.Query) (string, error) {
	// We need to get a unique set of array indexes for each path in the rego code.
	// That is tracked in this map.
	pathsToArrayIndexes := make(map[string]int)
	var fieldsAndMatchers []fieldMatchData

	for _, fieldQuery := range query.FieldQueries {
		field := fieldQuery.Field
		metaPathToField, found := r.fieldToMetaPathMap.Get(field)
		if !found {
			return "", fmt.Errorf("field %v not in object", field)
		}
		var constructedPath strings.Builder
		for i, elem := range metaPathToField {
			constructedPath.WriteString(elem.FieldName)
			if i == len(metaPathToField)-1 {
				// For the last element, we don't want to index into it, or add a "." at the end.
				break
			}
			if elem.Type.Kind() == reflect.Slice || elem.Type.Kind() == reflect.Array {
				pathKey := constructedPath.String()
				idx, ok := pathsToArrayIndexes[pathKey]
				if !ok {
					idx = len(pathsToArrayIndexes)
					pathsToArrayIndexes[pathKey] = idx
				}
				constructedPath.WriteString(fmt.Sprintf("[idx%d]", idx))
			}
			constructedPath.WriteString(".")
		}
		matchersForField, err := generateMatchersForField(fieldQuery, metaPathToField[len(metaPathToField)-1].Type)
		if err != nil {
			return "", fmt.Errorf("generating matchers for field query %+v: %w", fieldQuery, err)
		}
		fieldsAndMatchers = append(fieldsAndMatchers, fieldMatchData{
			matchers: matchersForField,
			name:     field,
			path:     constructedPath.String(),
		})
	}

	args := &mainProgramArgs{}
	for i := 0; i < len(pathsToArrayIndexes); i++ {
		args.IndexesToDeclare = append(args.IndexesToDeclare, i)
	}
	var funcLengths []int
	for _, matchData := range fieldsAndMatchers {
		for _, f := range matchData.matchers {
			args.Functions = append(args.Functions, f.functionCode)
		}
		funcLengths = append(funcLengths, len(matchData.matchers))
	}
	// We need to generate one rule for each cross product, since we are OR-ing between them.
	if err := runForEachCrossProduct(funcLengths, func(indexes []int) error {
		condition := condition{}
		for i, matchData := range fieldsAndMatchers {
			condition.Fields = append(condition.Fields, fieldInCondition{
				Name:     matchData.name,
				JSONPath: matchData.path,
				FuncName: matchData.matchers[indexes[i]].functionName,
			})
		}
		args.Conditions = append(args.Conditions, condition)
		return nil
	}); err != nil {
		return "", err
	}
	return generateMainProgram(args)
}

// This takes a list of array lengths, and invokes the func for every combination of the array indexes.
// For example, given array lengths [2, 3, 1],
// f will be called with
// [0, 0, 0]
// [0, 1, 0]
// [0, 2, 0]
// [1, 0, 0]
// [1, 1, 0]
// [1, 2, 0]
func runForEachCrossProduct(arrayLengths []int, f func([]int) error) error {
	for _, l := range arrayLengths {
		if l == 0 {
			return nil
		}
	}
	currentVal := make([]int, len(arrayLengths))
	for {
		if err := f(currentVal); err != nil {
			return err
		}
		idxToIncrement := 0
		for {
			if currentVal[idxToIncrement] < arrayLengths[idxToIncrement]-1 {
				currentVal[idxToIncrement]++
				break
			}
			if idxToIncrement == len(currentVal)-1 {
				return nil
			}
			currentVal[idxToIncrement] = 0
			idxToIncrement++
		}
	}
}
