package evaluator

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/query"
	"github.com/stackrox/stackrox/pkg/utils"
)

// An Evaluator evaluates an augmented object, and produces a result that has been filtered to linked matches.
type Evaluator interface {
	Evaluate(obj pathutil.AugmentedValue) (*Result, bool)
}

// A Factory knows how to create evaluators.
type Factory struct {
	fieldToMetaPaths *pathutil.FieldToMetaPathMap
	rootType         reflect.Type
}

// MustCreateNewFactory is like NewFactory, but panics if there's an error.
func MustCreateNewFactory(objMeta *pathutil.AugmentedObjMeta) Factory {
	f, err := NewFactory(objMeta)
	utils.CrashOnError(err)
	return f
}

// NewFactory returns a new Evaluator factory given metadata about the object and (optional) associated lifecycle stage.
func NewFactory(objMeta *pathutil.AugmentedObjMeta) (Factory, error) {
	fieldToMetaPaths, err := objMeta.MapSearchTagsToPaths()
	if err != nil {
		return Factory{}, errors.Wrap(err, "mapping search tags to paths")
	}
	return Factory{
		fieldToMetaPaths: fieldToMetaPaths,
		rootType:         objMeta.RootType(),
	}, nil
}

// GenerateEvaluator generates an Evaluator that will evaluate the given query
// on objects of the factory's type.
func (f *Factory) GenerateEvaluator(q *query.Query) (Evaluator, error) {
	internal, err := f.generateInternalEvaluator(q)
	if err != nil {
		return nil, err
	}
	return &panicCatchingEvaluator{internal: internal}, nil
}

type evaluatorFunc func(pathutil.AugmentedValue) (*Result, bool)

func (f evaluatorFunc) Evaluate(value pathutil.AugmentedValue) (*Result, bool) {
	return f(value)
}

type panicCatchingEvaluator struct {
	internal Evaluator
}

func (e *panicCatchingEvaluator) Evaluate(obj pathutil.AugmentedValue) (res *Result, matched bool) {
	// Ensure that we correctly catch calls to panic(nil) by
	// making sure that the defer is not called before the call to
	// e.internal.Evaluate returns successfully.
	panicked := true
	defer func() {
		// Panics can occur in evaluators, mainly due to incorrect uses of reflect.
		// This is always a programming error, but let's not panic in prod over it.
		if r := recover(); r != nil || panicked {
			utils.Should(errors.Errorf("panic running fieldEvaluator: %v", r))
			res = nil
			matched = false
		}
	}()
	res, matched = e.internal.Evaluate(obj)
	panicked = false
	return res, matched
}

func (f *Factory) generateInternalEvaluator(q *query.Query) (Evaluator, error) {
	// The field queries are implicitly a linked conjunction. This means that all field queries must match,
	// AND that their matches must be in the same object.
	// The notion of linking is a bit complicated -- the easiest way to get a sense of what it entails is to look
	// at the test cases in TestLinked.
	fieldEvaluators := make([]fieldEvaluator, 0, len(q.FieldQueries))
	for _, fq := range q.FieldQueries {
		eval, err := f.generateInternalEvaluatorForFieldQuery(fq)
		if err != nil {
			return nil, errors.Wrapf(err, "compiling field query: %v", fq)
		}

		fieldEvaluators = append(fieldEvaluators, eval)
	}

	switch len(fieldEvaluators) {
	case 0:
		return AlwaysTrue, nil
	default:
		return evaluatorFunc(func(value pathutil.AugmentedValue) (*Result, bool) {
			fieldsToPathsAndValues := make(map[string][]pathutil.PathAndValueHolder)
			for _, fieldEval := range fieldEvaluators {
				result, matches := fieldEval.Evaluate(value)
				if !matches {
					return nil, false
				}
				for field, matches := range result.Matches {
					for _, match := range matches {
						fieldsToPathsAndValues[field] = append(fieldsToPathsAndValues[field], match)
					}
				}
			}

			filteredResult, matched, err := pathutil.FilterMatchesToResults(fieldsToPathsAndValues)
			if err != nil {
				utils.Should(errors.Wrap(err, "filtering paths to linked matches"))
				return nil, false
			}
			if !matched {
				return nil, false
			}
			return &Result{filteredResult}, true
		}), nil
	}
}

// generateInternalEvaluatorForFieldQuery generates an internal fieldEvaluator for a specific field query.
func (f *Factory) generateInternalEvaluatorForFieldQuery(q *query.FieldQuery) (fieldEvaluator, error) {
	fieldPath, found := f.fieldToMetaPaths.Get(q.Field)
	if !found {
		return nil, errors.Errorf("invalid query: field %v unknown", q.Field)
	}

	baseType := fieldPath[len(fieldPath)-1].Type
	baseEvaluator, err := createBaseEvaluator(q.Field, baseType, q.Values, q.Negate, q.Operator, q.MatchAll)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid query %v", q)
	}

	pathEvaluator, err := f.wrapBaseEvaluatorWithPathTraversal(fieldPath, baseEvaluator)
	if err != nil {
		return nil, errors.Wrapf(err, "generating path traverser: %v", q)
	}
	return pathEvaluator, nil
}
