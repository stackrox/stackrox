package predicate

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/fieldmap"
	"github.com/stackrox/rox/pkg/utils"
)

// MergeResults merges predicate result into a single result
// If the results slice passed is of length 0, then it will return a nil result
func MergeResults(results ...*search.Result) *search.Result {
	if len(results) == 0 {
		return nil
	}
	res := search.NewResult()
	for _, r := range results {
		if r == nil {
			continue
		}
		for k, v := range r.Matches {
			res.Matches[k] = append(res.Matches[k], v...)
		}
	}
	return res
}

// Predicate represents a method that accesses data in some interface.
// NOTE: Predicates in general should not be compared, and doing so might cause a panic. An exception are the special
// predicates `AlwaysTrue` and `AlwaysFalse` -- any predicate can safely be compared against those special predicates
// to test whether its a constant predicate.
type Predicate interface {
	Evaluate(instance interface{}) (*search.Result, bool)
	Matches(instance interface{}) bool
}

// predicateFunc wraps a regular function as a predicate.
type predicateFunc func(instance interface{}) (*search.Result, bool)

func (f predicateFunc) Evaluate(instance interface{}) (*search.Result, bool) {
	return f(instance)
}

func (f predicateFunc) Matches(instance interface{}) bool {
	_, matches := f(instance)
	return matches
}

// Factory object stores the specs for each when walking the query.
type Factory struct {
	searchFields fieldmap.FieldMap
	searchPaths  wrappedOptionsMap
	exampleObj   interface{}
}

// NewFactory returns a new predicate factory for the type of the given object.
func NewFactory(prefix string, obj interface{}) Factory {
	return Factory{
		searchFields: fieldmap.MapSearchTagsToFieldPaths(obj),
		searchPaths: wrappedOptionsMap{
			optionsMap: search.Walk(v1.SearchCategory(-1), prefix, obj),
			prefix:     fmt.Sprintf("%s.", prefix),
		},
		exampleObj: obj,
	}
}

// ForCustomOptionsMap returns a factory with the same settings for the same object, but using a different search
// options map.
func (tb Factory) ForCustomOptionsMap(optsMap search.OptionsMap) Factory {
	return Factory{
		searchFields: tb.searchFields,
		searchPaths: wrappedOptionsMap{
			optionsMap: optsMap,
			prefix:     tb.searchPaths.prefix,
		},
		exampleObj: tb.exampleObj,
	}
}

// GeneratePredicate creates a predicate for the Predicate factories type that returns whether or not the input
// instance matches the query.
func (tb Factory) GeneratePredicate(query *v1.Query) (Predicate, error) {
	ip, err := tb.generatePredicateInternal(query)
	if err != nil {
		return nil, err
	}
	return wrapInternal(ip), nil
}

type internalPredicate interface {
	Evaluate(value reflect.Value) (*search.Result, bool)
}

type internalPredicateFunc func(reflect.Value) (*search.Result, bool)

func (f internalPredicateFunc) Evaluate(val reflect.Value) (*search.Result, bool) {
	return f(val)
}

func wrapInternal(ip internalPredicate) Predicate {
	if ip == alwaysTrue {
		return AlwaysTrue
	}
	if ip == alwaysFalse {
		return AlwaysFalse
	}
	return predicateFunc(func(in interface{}) (*search.Result, bool) {
		val := reflect.ValueOf(in)
		return ip.Evaluate(val)
	})
}

func (tb Factory) generatePredicateInternal(query *v1.Query) (internalPredicate, error) {
	if query == nil || query.GetQuery() == nil {
		return alwaysTrue, nil
	}
	switch query.GetQuery().(type) {
	case *v1.Query_Disjunction:
		return tb.or(query.GetDisjunction())
	case *v1.Query_Conjunction:
		return tb.and(query.GetConjunction())
	case *v1.Query_BooleanQuery:
		return tb.boolean(query.GetBooleanQuery())
	case *v1.Query_BaseQuery:
		return tb.base(query.GetBaseQuery())
	default:
		return nil, fmt.Errorf("unrecognized query type: %T", query.GetQuery())
	}
}

func (tb Factory) or(q *v1.DisjunctionQuery) (internalPredicate, error) {
	ret := make([]internalPredicate, 0, len(q.GetQueries()))
	for _, dis := range q.GetQueries() {
		next, err := tb.generatePredicateInternal(dis)
		if err != nil {
			return nil, err
		}
		ret = append(ret, next)
	}

	return orOf(ret...), nil
}

func (tb Factory) and(q *v1.ConjunctionQuery) (internalPredicate, error) {
	ret := make([]internalPredicate, 0, len(q.GetQueries()))
	for _, dis := range q.GetQueries() {
		next, err := tb.generatePredicateInternal(dis)
		if err != nil {
			return nil, err
		}
		ret = append(ret, next)
	}

	return andOf(ret...), nil
}

func (tb Factory) boolean(q *v1.BooleanQuery) (internalPredicate, error) {
	must, err := tb.and(q.GetMust())
	if err != nil {
		return nil, err
	}

	mustNot, err := tb.or(q.GetMustNot())
	if err != nil {
		return nil, err
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		mustRes, mustMatch := must.Evaluate(instance)
		if !mustMatch {
			return nil, false
		}
		mustNotRes, mustNotMatch := mustNot.Evaluate(instance)
		if mustNotMatch {
			return nil, false
		}
		return MergeResults(mustRes, mustNotRes), true
	}), nil
}

func (tb Factory) base(q *v1.BaseQuery) (internalPredicate, error) {
	switch q.GetQuery().(type) {
	case *v1.BaseQuery_DocIdQuery:
		return tb.docID(q.GetDocIdQuery())
	case *v1.BaseQuery_MatchNoneQuery:
		return tb.matchNone(q.GetMatchNoneQuery())
	case *v1.BaseQuery_MatchFieldQuery:
		return tb.match(q.GetMatchFieldQuery())
	case *v1.BaseQuery_MatchLinkedFieldsQuery:
		return tb.matchLinked(q.GetMatchLinkedFieldsQuery())
	default:
		return nil, fmt.Errorf("cannot handle base query of type %T", q.GetQuery())
	}
}

func (tb Factory) docID(_ *v1.DocIDQuery) (internalPredicate, error) {
	return nil, errors.New("query predicates do not support DocID query types as DocIDs only exist in the index")
}

func (tb Factory) matchNone(_ *v1.MatchNoneQuery) (internalPredicate, error) {
	return alwaysFalse, nil
}

func (tb Factory) match(q *v1.MatchFieldQuery) (internalPredicate, error) {
	fp := tb.searchFields.Get(strings.ToLower(q.GetField()))
	if fp == nil {
		return alwaysTrue, nil
	}
	sp, ok := tb.searchPaths.Get(q.GetField())
	if !ok {
		return tb.createPredicate("", fp, q.GetValue())
	}
	return tb.createPredicate(sp.GetFieldPath(), fp, q.GetValue())
}

func (tb Factory) matchLinked(q *v1.MatchLinkedFieldsQuery) (internalPredicate, error) {
	// Find the longest common path with all of the linked fields.
	var commonPath fieldmap.FieldPath
	for _, fieldQuery := range q.GetQuery() {
		path := tb.searchFields.Get(fieldQuery.GetField())
		if path == nil {
			return alwaysTrue, nil
		}
		if commonPath == nil {
			commonPath = path[:len(path)-1]
		} else {
			for idx, field := range path {
				if idx > len(commonPath)-1 {
					break
				}
				if commonPath[idx].Name != field.Name {
					commonPath = commonPath[:idx]
					break
				}
			}
		}
	}

	predRootTy := reflect.TypeOf(tb.exampleObj)
	if len(commonPath) > 0 {
		predRootTy = commonPath[len(commonPath)-1].Type
		switch predRootTy.Kind() {
		case reflect.Array, reflect.Slice:
			predRootTy = predRootTy.Elem() // root type may be a pointer, but not a slice.
		}
	}

	// Produce a predicate for each of the fields. Use the non common path.
	var preds []internalPredicate
	for _, fieldQuery := range q.GetQuery() {
		path := tb.searchFields.Get(fieldQuery.GetField())
		if path == nil {
			return alwaysTrue, nil
		}
		var fieldPath string
		searchField, ok := tb.searchPaths.Get(fieldQuery.GetField())
		if ok {
			fieldPath = searchField.GetFieldPath()
		}

		pred, err := tb.createPredicateWithRootType(predRootTy, fieldPath, path[len(commonPath):], fieldQuery.GetValue())
		if err != nil {
			return nil, err
		}
		preds = append(preds, pred)
	}

	// Package all the of predicates as an AND on the common path.
	var linked internalPredicate
	if len(commonPath) > 0 {
		var err error
		linked, err = createLinkedNestedPredicate(commonPath[len(commonPath)-1].Type, preds...)
		if err != nil {
			return nil, err
		}
	} else {
		linked = andOf(preds...)
	}

	return createPathPredicate(reflect.TypeOf(tb.exampleObj), commonPath, linked)
}

func (tb Factory) createPredicate(fullPath string, path fieldmap.FieldPath, value string) (internalPredicate, error) {
	return tb.createPredicateWithRootType(reflect.TypeOf(tb.exampleObj), fullPath, path, value)
}

func (tb Factory) createPredicateWithRootType(rootTy reflect.Type, fullPath string, path fieldmap.FieldPath, value string) (internalPredicate, error) {
	// Create the predicate for the search field value.
	pred, err := createBasePredicate(fullPath, path[len(path)-1].Type, value)
	if err != nil {
		return nil, err
	}

	// Create a wrapper predicate which traces the field path down to the value.
	pred, err = createPathPredicate(rootTy, path, pred)
	if err != nil {
		return nil, err
	}
	return pred, nil
}

// Combinatorial helpers.
/////////////////////////

func orOf(preds ...internalPredicate) internalPredicate {
	filtered := preds[:0]
	for _, pred := range preds {
		if pred == alwaysTrue {
			return alwaysTrue
		}
		if pred == alwaysFalse {
			continue
		}
		filtered = append(filtered, pred)
	}

	if len(filtered) == 0 {
		return alwaysFalse
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		var results []*search.Result
		for _, pred := range filtered {
			if res, match := pred.Evaluate(instance); match {
				results = append(results, res)
			}
		}
		if len(results) > 0 {
			return MergeResults(results...), true
		}
		return nil, false
	})
}

func andOf(preds ...internalPredicate) internalPredicate {
	filtered := preds[:0]
	for _, pred := range preds {
		if pred == alwaysTrue {
			continue
		}
		if pred == alwaysFalse {
			return alwaysFalse
		}
		filtered = append(filtered, pred)
	}

	if len(filtered) == 0 {
		return alwaysTrue
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		var results []*search.Result
		for _, pred := range filtered {
			result, ok := pred.Evaluate(instance)
			if !ok {
				return nil, false
			}
			results = append(results, result)
		}
		return MergeResults(results...), true
	})
}

// Recursive predicate manufacturing from the input field path.
///////////////////////////////////////////////////////////////

func createPathPredicate(parentType reflect.Type, path fieldmap.FieldPath, pred internalPredicate) (internalPredicate, error) {
	if len(path) == 0 {
		return pred, nil
	}

	// If not, recursively go down to the base.
	child, err := createPathPredicate(path[0].Type, path[1:], pred)
	if err != nil {
		return nil, err
	}

	// Wrap the predicate in field access.
	return createNestedPredicate(parentType, path[0], child)
}

func createNestedPredicate(parentType reflect.Type, field reflect.StructField, pred internalPredicate) (internalPredicate, error) {
	switch parentType.Kind() {
	case reflect.Array, reflect.Slice:
		return createSliceNestedPredicate(parentType, field, pred)
	case reflect.Ptr:
		return createPtrNestedPredicate(parentType, field, pred)
	case reflect.Map:
		return createMapNestedPredicate(parentType, field, pred)
	case reflect.Struct:
		return createStructFieldNestedPredicate(field, parentType, pred), nil
	case reflect.Interface:
		return createInterfaceFieldNestedPredicate(field, pred), nil
	default:
		return alwaysFalse, fmt.Errorf("cannot follow: %+v", field)
	}
}

// Complex type predicates.
////////////////////////////

func createSliceNestedPredicate(parentType reflect.Type, field reflect.StructField, pred internalPredicate) (internalPredicate, error) {
	nested, err := createNestedPredicate(parentType.Elem(), field, pred)
	if err != nil {
		return nil, err
	}

	if nested == alwaysFalse {
		return alwaysFalse, nil
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsNil() || instance.IsZero() {
			return nil, false
		}
		var results []*search.Result
		length := instance.Len()
		for i := 0; i < length; i++ {
			idx := instance.Index(i)
			if res, matches := nested.Evaluate(idx); matches {
				results = append(results, res)
			}
		}
		if len(results) > 0 {
			return MergeResults(results...), true
		}
		return nil, false
	}), nil
}

func createPtrNestedPredicate(parentType reflect.Type, field reflect.StructField, pred internalPredicate) (internalPredicate, error) {
	nested, err := createNestedPredicate(parentType.Elem(), field, pred)
	if err != nil {
		return nil, err
	}

	if nested == alwaysFalse {
		return alwaysFalse, nil
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsNil() { // Need to special handle pointers to nil. Good ole typed nils.
			return nil, false
		}
		elem := instance.Elem()
		if res, matches := nested.Evaluate(elem); matches {
			return res, true
		}
		return nil, false
	}), nil
}

func createMapNestedPredicate(parentType reflect.Type, field reflect.StructField, pred internalPredicate) (internalPredicate, error) {
	nestedKey, err := createNestedPredicate(parentType.Key(), field, pred)
	if err != nil {
		return nil, err
	}
	parentTypeElem := parentType.Elem()
	nestedElem, err := createNestedPredicate(parentTypeElem, field, pred)
	if err != nil {
		return nil, err
	}

	if nestedKey == alwaysFalse && nestedElem == alwaysFalse {
		return alwaysFalse, nil
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsNil() || instance.IsZero() {
			return nil, false
		}
		for _, key := range instance.MapKeys() {
			valueAt := instance.MapIndex(key)
			res, match := nestedKey.Evaluate(key)
			if match {
				return res, true
			}
			res, match = nestedElem.Evaluate(valueAt)
			if match {
				return res, true
			}
		}
		return nil, false
	}), nil
}

func nilCheck(f reflect.Value) bool {
	switch f.Kind() {
	// Don't return nil for nil Reflect.Maps.  Map base predicates should operate on nil maps
	case reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return f.IsNil()
	}
	return false
}

var (
	imageScanPtrType = reflect.TypeOf((*storage.ImageScan)(nil))
)

func createStructFieldNestedPredicate(field reflect.StructField, structTy reflect.Type, pred internalPredicate) internalPredicate {
	if pred == alwaysFalse {
		return alwaysFalse
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.Type() != structTy {
			utils.Should(errors.Errorf("unexpected type mismatch for nested struct field: got %s, expected %s", instance.Type(), structTy))
			return nil, false
		}
		nextValue := instance.FieldByIndex(field.Index)
		if !nilCheck(nextValue) || nextValue.Type() == timestampPtrType {
			return pred.Evaluate(nextValue)
		}
		// Special-case image scans, replacing a nil scan with an empty scan.
		// Note: the special-casing is done in this hacky way to minimize the changes for cherry-picking,
		// and because predicates are going away.
		if nextValue.Type() == imageScanPtrType {
			return pred.Evaluate(reflect.New(imageScanPtrType.Elem()))
		}
		// The value is nil, and not one of the special-cases where a nil value should be evaluated.
		return nil, false
	})
}

func createInterfaceFieldNestedPredicate(field reflect.StructField, pred internalPredicate) internalPredicate {
	if pred == alwaysFalse {
		return alwaysFalse
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsNil() || instance.IsZero() {
			return nil, false
		}
		concrete := instance.Elem()
		if concrete.Type().Kind() == reflect.Ptr {
			concrete = concrete.Elem()
		}
		if concrete.Type().Kind() != reflect.Struct {
			return nil, false
		}
		nextValue := concrete.FieldByName(field.Name)
		if nextValue.IsZero() {
			return nil, false // Field either does not exist, or wasn't populated.
		}
		return pred.Evaluate(nextValue)
	})
}
