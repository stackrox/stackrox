package mapeval

import (
	"container/heap"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/regexutils"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

const (
	// DisjunctionMarker marks a disjunction between conjunction groups in a query.
	DisjunctionMarker = ";\t"
	// ConjunctionMarker marks a conjunction between simpler key=value pairs in a query.
	ConjunctionMarker = ",\t"
	// ShouldNotMatchMarker adds a marker that indicates that a key=value pair should not be matched in a map.
	ShouldNotMatchMarker = "!\t"
)

type kvConstraint struct {
	key   regexutils.WholeStringMatcher
	value regexutils.WholeStringMatcher
}

type conjunctionGroupConstraint struct {
	shouldNotMatch []*kvConstraint
	shouldMatch    []*kvConstraint
}

func (c *conjunctionGroupConstraint) newState() *conjunctionMatchState {
	return &conjunctionMatchState{
		constraints:        c,
		shouldMatchResults: make([]bool, len(c.shouldMatch)),
		matchedPairs:       make(map[*KeyValue]struct{}),
	}
}

type conjunctionMatchState struct {
	constraints        *conjunctionGroupConstraint
	shouldMatchResults []bool
	matchedPairs       map[*KeyValue]struct{}

	shouldNotMatchFailed bool
}

func (c *conjunctionMatchState) checkAgainstNewKV(kv *KeyValue) {
	if c.shouldNotMatchFailed {
		return
	}
	for _, query := range c.constraints.shouldNotMatch {
		if valueMatchesRegex(query.key, kv.Key) && valueMatchesRegex(query.value, kv.Value) {
			c.shouldNotMatchFailed = true
			return
		}
	}

	var kvMatchedShouldMatch bool
	for i, query := range c.constraints.shouldMatch {
		if kvMatchedShouldMatch && c.shouldMatchResults[i] {
			continue
		}
		if valueMatchesRegex(query.key, kv.Key) && valueMatchesRegex(query.value, kv.Value) {
			kvMatchedShouldMatch = true
			c.shouldMatchResults[i] = true
		}
	}
	if kvMatchedShouldMatch {
		c.matchedPairs[kv] = struct{}{}
	}
}

func (c *conjunctionMatchState) matched() (map[*KeyValue]struct{}, bool) {
	if c.shouldNotMatchFailed {
		return nil, false
	}
	for _, res := range c.shouldMatchResults {
		if !res {
			return nil, false
		}
	}
	return c.matchedPairs, true
}

// KeyValue is a basic abstraction for matched key values.
type KeyValue struct {
	Key   string
	Value string
}

type matchedGroup struct {
	ShouldNotMatch []*KeyValue
	ShouldMatch    []*KeyValue
}

// KeyValueSlice is a slice of KeyValues.
// It is defined as a type to implement heap.Interface.
// It is a max-heap.
type KeyValueSlice []*KeyValue

// Len implements heap.Interface.
func (k *KeyValueSlice) Len() int {
	return len(*k)
}

// Less implements heap.Interface.
func (k *KeyValueSlice) Less(i, j int) bool {
	return (*k)[i].Key > (*k)[j].Key
}

// Swap implements heap.Interface.
func (k *KeyValueSlice) Swap(i, j int) {
	(*k)[j], (*k)[i] = (*k)[i], (*k)[j]
}

// Push implements heap.Interface.
func (k *KeyValueSlice) Push(x interface{}) {
	*k = append(*k, x.(*KeyValue))
}

// Pop implements heap.Interface.
func (k *KeyValueSlice) Pop() interface{} {
	length := len(*k)
	ret := (*k)[length-1]
	*k = (*k)[0 : length-1]
	return ret
}

// MatcherResults is the results returned from the matcher.
type MatcherResults struct {
	MatchingKeyValues map[*KeyValue]struct{}
	KeyValues         heap.Interface
	NumElements       int
	// Groups joined that disjunction, that are satisfied.
	Groups []*matchedGroup
	// MapVals returns 'numValuesToReturn' values from the map.
	MapVals []*KeyValue
}

func newMatcherResults() *MatcherResults {
	kvs := make(KeyValueSlice, 0)
	heap.Init(&kvs)
	return &MatcherResults{KeyValues: &kvs}
}

func regexpMatcherFromString(val string) (regexutils.WholeStringMatcher, error) {
	if val == "" {
		return nil, nil
	}

	return regexutils.CompileWholeStringMatcher(val, regexutils.Flags{CaseInsensitive: true})
}

func convertConjunctionPairsToGroupConstraint(conjunctionPairsStr string) (*conjunctionGroupConstraint, error) {
	ps := strings.Split(conjunctionPairsStr, ConjunctionMarker)
	if len(ps) == 0 {
		return nil, nil
	}

	conjunctionGroup := &conjunctionGroupConstraint{}
	for _, p := range ps {
		if !strings.Contains(p, "=") {
			return nil, errors.Errorf("Invalid key-value expression: %s", p)
		}

		p, shouldNotMatchQuery := stringutils.MaybeTrimPrefix(p, ShouldNotMatchMarker)
		k, v := stringutils.Split2(p, "=")
		key, err := regexpMatcherFromString(k)
		if err != nil {
			return nil, errors.Wrap(err, "invalid key")
		}

		value, err := regexpMatcherFromString(v)
		if err != nil {
			return nil, errors.Wrap(err, "invalid value")
		}

		ele := &kvConstraint{value: value, key: key}
		if shouldNotMatchQuery {
			conjunctionGroup.shouldNotMatch = append(conjunctionGroup.shouldNotMatch, ele)
		} else {
			conjunctionGroup.shouldMatch = append(conjunctionGroup.shouldMatch, ele)
		}
	}

	return conjunctionGroup, nil
}

func valueMatchesRegex(matcher regexutils.WholeStringMatcher, val string) bool {
	return matcher == nil || matcher.MatchWholeString(val)
}

// Matcher returns a matcher for a map against a query string. The returned matcher also accepts an int for the number of
// values in the map to return.
func Matcher(value string, typ reflect.Type) (func(*reflect.MapIter, int) (*MatcherResults, bool), error) {
	if typ.Key().Kind() != reflect.String || typ.Elem().Kind() != reflect.String {
		return nil, errors.Errorf("invalid typ %v: only map[string]string is supported", typ)
	}
	// The format for the query is taken to be a disjunction of groups.
	// A group is composed of conjunction of shouldNotMatch and shouldMatch (k,*) (*,v) (k,v) pairs.
	// A shouldMatch pair returns true if it is contained in the map.
	// A shouldNotMatch pair returns true if it is not present in the map.
	// Disjunction is marked by semicolons, Conjunction by commas
	// Should not match groups are preceded by a ! marker, and key value pairs appear as k=v
	// Eg: !a=, b=1; c=2;
	// The above expression is composed of two groups:
	// The first group implies that the map matches if key 'a' is absent, and b=1 is present.
	// The second group implies that the map matches if c=2 is present.
	var conjunctionGroupConstraints []*conjunctionGroupConstraint
	for _, conjunctionPairsStr := range strings.Split(value, DisjunctionMarker) {
		cg, err := convertConjunctionPairsToGroupConstraint(conjunctionPairsStr)
		if err != nil {
			return nil, err
		}

		if cg == nil {
			continue
		}

		conjunctionGroupConstraints = append(conjunctionGroupConstraints, cg)
	}

	return func(iter *reflect.MapIter, maxValues int) (*MatcherResults, bool) {
		conjunctionGroupStates := make([]*conjunctionMatchState, 0, len(conjunctionGroupConstraints))
		for _, c := range conjunctionGroupConstraints {
			conjunctionGroupStates = append(conjunctionGroupStates, c.newState())
		}
		res := newMatcherResults()
		for iter.Next() {
			// Only string type key, value are allowed.
			// It will only happen in the event of a programming error anyway, since we check the map
			// type above and return an error.
			kv := &KeyValue{iter.Key().Interface().(string), iter.Value().Interface().(string)}

			for _, state := range conjunctionGroupStates {
				state.checkAgainstNewKV(kv)
			}

			if res.KeyValues.Len() < maxValues {
				heap.Push(res.KeyValues, kv)
			} else {
				largestElem := heap.Pop(res.KeyValues).(*KeyValue)
				// At this point, the heap contains the `maxValues` smallest (in terms of alphabetical order of key)
				// kv pairs seen so far. If kv is smaller than the largest of these, then it will be in the new heap,
				// and the old largestElem must go.
				if kv.Key < largestElem.Key {
					heap.Push(res.KeyValues, kv)
				} else {
					heap.Push(res.KeyValues, largestElem)
				}
			}
			res.NumElements++
		}

		var atLeastOneMatched bool
		for _, state := range conjunctionGroupStates {
			matchedPairs, matched := state.matched()
			if matched {
				atLeastOneMatched = true
				if len(matchedPairs) > 0 {
					if res.MatchingKeyValues == nil {
						res.MatchingKeyValues = make(map[*KeyValue]struct{}, len(matchedPairs))
					}
					// This code depends on the fact that we have exactly one *KeyValue object for each
					// element of the map, that we pass around everywhere.
					for k := range matchedPairs {
						res.MatchingKeyValues[k] = struct{}{}
					}
				}
			}
		}

		return res, atLeastOneMatched
	}, nil
}
