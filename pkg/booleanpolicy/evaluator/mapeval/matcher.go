package mapeval

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/regexutils"
	"github.com/stackrox/rox/pkg/stringutils"
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
	key   *regexp.Regexp
	value *regexp.Regexp
}

type groupConstraint struct {
	shouldNotMatch []*kvConstraint
	shouldMatch    []*kvConstraint
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

// MatcherResults is the results returned from the matcher.
type MatcherResults struct {
	// Groups joined that disjunction, that are satisfied.
	Groups []*matchedGroup
	// MapVals returns 'numValuesToReturn' values from the map.
	MapVals []*KeyValue
}

func assignRegExpFromString(val string) (*regexp.Regexp, error) {
	if val == "" {
		return nil, nil
	}

	return regexp.Compile(val)
}

func convertConjunctionPairsToGroupConstraint(conjunctionPairsStr string) (*groupConstraint, error) {
	ps := strings.Split(conjunctionPairsStr, ConjunctionMarker)
	if len(ps) == 0 {
		return nil, nil
	}

	conjunctionGroup := &groupConstraint{}
	for _, p := range ps {
		if !strings.Contains(p, "=") {
			return nil, errors.Errorf("Invalid key-value expression: %s", p)
		}

		p, shouldNotMatchQuery := stringutils.MaybeTrimPrefix(p, ShouldNotMatchMarker)
		k, v := stringutils.Split2(p, "=")
		key, err := assignRegExpFromString(k)
		if err != nil {
			return nil, errors.Wrap(err, "invalid key")
		}

		value, err := assignRegExpFromString(v)
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

func valueMatchesRequest(req *regexp.Regexp, val string) bool {
	return req == nil || regexutils.MatchWholeString(req, val)
}

func verifyAgainstCG(gE *groupConstraint, kvMatchStates map[*kvConstraint][]*KeyValue, key, value string) {
	for _, r := range gE.shouldNotMatch {
		if valueMatchesRequest(r.key, key) && valueMatchesRequest(r.value, value) {
			kvMatchStates[r] = append(kvMatchStates[r], &KeyValue{
				Key:   key,
				Value: value,
			})
		}
	}

	for _, d := range gE.shouldMatch {
		if valueMatchesRequest(d.key, key) && valueMatchesRequest(d.value, value) {
			kvMatchStates[d] = append(kvMatchStates[d], &KeyValue{
				Key:   key,
				Value: value,
			})
		}
	}
}

func matchesCG(gE *groupConstraint, kvMatchStates map[*kvConstraint][]*KeyValue) *matchedGroup {
	for _, r := range gE.shouldNotMatch {
		if len(kvMatchStates[r]) > 0 {
			return nil
		}
	}
	// All shouldNotMatch requirements failed at this point.

	for _, d := range gE.shouldMatch {
		if len(kvMatchStates[d]) == 0 {
			return nil
		}
	}
	// Now, all shouldMatch requirements failed at this point, so this map matches this particular conjunction
	// group.
	res := &matchedGroup{}
	for _, r := range gE.shouldNotMatch {
		key := ""
		if r.key != nil {
			key = r.key.String()
		}

		val := ""
		if r.value != nil {
			val = r.value.String()
		}

		res.ShouldNotMatch = append(res.ShouldNotMatch, &KeyValue{
			Key:   key,
			Value: val,
		})
	}

	for _, d := range gE.shouldMatch {
		res.ShouldMatch = append(res.ShouldMatch, kvMatchStates[d]...)
	}

	return res
}

// Matcher returns a matcher for a map against a query string. It also returns upto N values of the map.
func Matcher(value string) (func(*reflect.MapIter, int) (*MatcherResults, bool), error) {
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
	var disjunctionGroups []*groupConstraint
	for _, conjunctionPairsStr := range strings.Split(value, DisjunctionMarker) {
		cg, err := convertConjunctionPairsToGroupConstraint(conjunctionPairsStr)
		if err != nil {
			return nil, err
		}

		if cg == nil {
			continue
		}

		disjunctionGroups = append(disjunctionGroups, cg)
	}

	return func(iter *reflect.MapIter, numValuesToReturn int) (*MatcherResults, bool) {
		kvMatchStates := make(map[*kvConstraint][]*KeyValue)
		var res *MatcherResults
		count := 0
		for iter.Next() {
			k, v := iter.Key(), iter.Value()
			// Only string type key, value are allowed.
			key, ok := k.Interface().(string)
			if !ok {
				return nil, false
			}

			value, ok := v.Interface().(string)
			if !ok {
				return nil, false
			}

			for _, cg := range disjunctionGroups {
				verifyAgainstCG(cg, kvMatchStates, key, value)
			}

			if numValuesToReturn > count {
				if res == nil {
					res = &MatcherResults{}
				}

				res.MapVals = append(res.MapVals, &KeyValue{Key: key, Value: value})
				count++
			}
		}

		for _, cg := range disjunctionGroups {
			// Apply disjunction and return true if any group is true.
			if mg := matchesCG(cg, kvMatchStates); mg != nil {
				if res == nil {
					res = &MatcherResults{}
				}

				res.Groups = append(res.Groups, mg)
			}
		}

		return res, res != nil && len(res.Groups) > 0
	}, nil
}
