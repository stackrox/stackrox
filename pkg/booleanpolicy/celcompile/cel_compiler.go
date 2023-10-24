package celcompile

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type celCompilerForType struct {
	fieldToMetaPathMap *pathutil.FieldToMetaPathMap
}

// A CelCompiler compiles a CEL-based evaluator for the given query.
type CelCompiler interface {
	CompileCelBasedEvaluator(query *query.Query) (evaluator.Evaluator, error)
}

type celBasedEvaluator struct {
	q      cel.Program
	module string // For debug
}

// convertBindingToResult converts a set of variable bindings to a result.
// It has to do a bunch of type assertions, since cel can return arbitrary values.
// We know that our cel programs are constructed to return map[string][]interface{},
// so this takes advantage of that to traverse them. It also converts each returned value
// into a string.
func convertBindingToResult(binding map[string][]interface{}) (m map[string][]string, err error) {
	panicked := true
	defer func() {
		if r := recover(); r != nil || panicked {
			err = fmt.Errorf("panic running evaluator: %v", r)
		}
	}()
	m = make(map[string][]string)
	for k, v := range binding {
		vAsInterfaceSlice := v
		vAsString := make([]string, 0, len(vAsInterfaceSlice))
		for _, val := range vAsInterfaceSlice {
			vAsString = append(vAsString, fmt.Sprintf("%v", val))
		}
		m[k] = vAsString
	}
	panicked = false
	return m, nil
}

func (r *celBasedEvaluator) Evaluate(obj *pathutil.AugmentedObj) (*evaluator.Result, bool) {
	value, err := obj.GetFullValue()
	if err != nil {
		utils.Should(err)
		return nil, false
	}
	val, _, err := r.q.Eval(map[string]interface{}{"obj": value})
	if err != nil {
		jsonData, _ := val.ConvertToNative(reflect.TypeOf(&structpb.Value{}))
		out := protojson.Format(jsonData.(*structpb.Value))
		log.Errorf("Compiled module %s\nOutput: %s\nerror: %s", r.module, out, err)
		utils.Should(err)
		return nil, false
	}

	result, err := val.ConvertToNative(reflect.TypeOf([]map[string][]any{}))
	if err != nil {
		err = fmt.Errorf("invalid result: %+v", result)
		utils.Should(err)
		return nil, false
	}
	res := &evaluator.Result{}
	for _, binding := range result.([]map[string][]interface{}) {
		match, err := convertBindingToResult(binding)
		if err != nil {
			err = fmt.Errorf("invalid result: %+v", result)
			utils.Should(err)
			return nil, false
		}
		res.Matches = append(res.Matches, match)
	}

	return res, len(res.Matches) != 0
}

// MustCreateCompiler is a wrapper around CreateCelCompiler that panics if there's an error.
func MustCreateCompiler(objMeta *pathutil.AugmentedObjMeta) CelCompiler {
	r, err := CreateCelCompiler(objMeta)
	utils.Must(err)
	return r
}

// CreateCelCompiler creates a CEL compiler for the given object meta.
func CreateCelCompiler(objMeta *pathutil.AugmentedObjMeta) (CelCompiler, error) {
	fieldToMetaPathMap, err := objMeta.MapSearchTagsToPaths()
	if err != nil {
		return nil, err
	}
	return &celCompilerForType{fieldToMetaPathMap: fieldToMetaPathMap}, nil
}

func (r *celCompilerForType) CompileCelBasedEvaluator(query *query.Query) (evaluator.Evaluator, error) {
	module, err := r.compileCel(query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile cel: %w", err)
	}
	module = CelPrettyPrint(module)
	// log.Infof("Compiled: \n%s", module)
	prg, err := compile(module)
	if err != nil {
		return nil, err
	}
	return &celBasedEvaluator{q: prg, module: module}, nil
}

func (r *celCompilerForType) compileCel(query *query.Query) (string, error) {
	args := &mainProgramArgs{}
	args.Root = MatchField{
		VarName:   "obj",
		Path:      "obj",
		CheckCode: "true",
	}
	pathsToAccessVariable := map[string]*MatchField{"obj": &args.Root}

	for _, fieldQuery := range query.FieldQueries {
		field := fieldQuery.Field
		metaPathToField, found := r.fieldToMetaPathMap.Get(field)
		if !found {
			return "", fmt.Errorf("field %v not in object", field)
		}
		var constructedPath strings.Builder
		var currentPath strings.Builder
		constructedPath.WriteString("obj.")
		currentPath.WriteString("obj")
		parent := &args.Root
		for i, elem := range metaPathToField {
			constructedPath.WriteString(elem.FieldName)
			currentPath.WriteString("." + elem.FieldName)
			if i == len(metaPathToField)-1 {
				// For the last element, we don't want to index into it, or add a "." at the end.
				break
			}
			if elem.Type.Kind() == reflect.Slice || elem.Type.Kind() == reflect.Array {
				pathKey := constructedPath.String()
				mf, ok := pathsToAccessVariable[pathKey]
				if !ok {
					checkCode := generateCheckCode(currentPath.String())
					if checkCode == "" {
						checkCode = "true"
					}
					mf = &MatchField{
						VarName:   currentPath.String(),
						Path:      constructedPath.String(),
						CheckCode: checkCode,
					}
					parent.Children = append(parent.Children, mf)
					pathsToAccessVariable[pathKey] = mf
				}
				parent = mf
				currentPath.Reset()
				currentPath.WriteString("k")
			}
			constructedPath.WriteString(".")
		}
		code, err := generateMatchCodeForField(fieldQuery, metaPathToField[len(metaPathToField)-1].Type, currentPath.String())
		if err != nil {
			return "", fmt.Errorf("generating matchers for field query %+v: %w", fieldQuery, err)
		}
		checkCode := generateCheckCode(currentPath.String())
		if checkCode != "" {
			code = fmt.Sprintf("%s && (%s)", checkCode, code)
		} else {
			checkCode = "true"
		}
		mf := &MatchField{
			VarName:    currentPath.String(),
			SearchName: fieldQuery.Field,
			MatchCode:  code,
			IsLeaf:     true,
			Path:       constructedPath.String(),
			CheckCode:  checkCode,
		}
		parent.Children = append(parent.Children, mf)
	}
	return generateMainProgram(args)
}
