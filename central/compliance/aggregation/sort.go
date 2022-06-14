package aggregation

import (
	"sort"
	"strconv"
	"strings"

	"github.com/stackrox/stackrox/generated/storage"
)

func sortAggregations(results []*storage.ComplianceAggregation_Result) {
	sort.SliceStable(results, func(a, b int) bool {
		return aBeforeB(results[a].GetAggregationKeys(), results[b].GetAggregationKeys())
	})
}

func aBeforeB(a, b []*storage.ComplianceAggregation_AggregationKey) bool {
	// Get the length of the smallest aggregtion key set.
	var minLen int
	if len(a) < len(b) {
		minLen = len(a)
	} else {
		minLen = len(b)
	}

	// Try to choose a or b based on scope. Lowest scope type works first.
	for i := 0; i < minLen; i++ {
		if a[i].GetScope() < b[i].GetScope() {
			return true
		} else if a[i].GetScope() > b[i].GetScope() {
			return false
		}
	}

	// No lower scope, so is one more scoped than the other?
	if len(a) > minLen {
		return false
	} else if len(b) > minLen {
		return true
	}

	// Same exact scopes, so we have to choose order based on ids.
	for i := 0; i < minLen; i++ {
		var cmp int
		if a[i].GetScope() == storage.ComplianceAggregation_CONTROL {
			// We want to use version string comparison for control ids (1_20_a > 1_2_a)
			cmp = versionCompare(a[i].GetId(), b[i].GetId())
		} else {
			cmp = strings.Compare(a[i].GetId(), b[i].GetId())
		}
		if cmp < 0 {
			return true
		} else if cmp > 0 {
			return false
		}
	}

	// This should never happen, as it means the scopes and ids are all the same.
	return true
}

// Version string like compare.
// 1_20_a > 1_2_a
func versionCompare(a, b string) int {
	as := strings.Split(a, "_")
	bs := strings.Split(b, "_")
	for i := 0; i < len(as) && i < len(bs); i++ {
		av, aErr := strconv.Atoi(as[i])
		bv, bErr := strconv.Atoi(bs[i])
		if aErr == nil && bErr == nil { // Both are numbers.
			if av < bv {
				return -1
			} else if av > bv {
				return 1
			}
		} else if aErr == nil && bErr != nil { // a is a number, b is a word.
			return -1
		} else if aErr != nil && bErr == nil { // b is a number, a is a word.
			return 1
		} else if aErr != nil && bErr != nil { // Both are words.
			cmp := strings.Compare(as[i], bs[i])
			if cmp != 0 {
				return cmp
			}
		}
	}
	if len(a) < len(b) {
		return -1
	} else if len(b) > len(a) {
		return 1
	}
	return 0
}
