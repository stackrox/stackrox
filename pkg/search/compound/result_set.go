package compound

import (
	"sort"
	"strings"

	"github.com/stackrox/rox/pkg/search"
)

type resultSet struct {
	results []search.Result
	order   map[string]int
}

func newResultSet(results []search.Result, ordered bool) resultSet {
	var order map[string]int
	if ordered {
		order = make(map[string]int, len(results))
		for idx, res := range results {
			order[res.ID] = idx
		}
	}
	// Sort by id for searching and combining.
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	return resultSet{
		results: results,
		order:   order,
	}
}

func (rs resultSet) intersect(other resultSet) resultSet {
	minLen := len(rs.results)
	if len(other.results) < minLen {
		minLen = len(other.results)
	}

	order := rs.order
	if other.order != nil {
		order = other.order
	}

	newResults := make([]search.Result, 0, minLen)
	thatIdx := 0
	thisIdx := 0
	thisInBounds := thisIdx < len(rs.results)
	thatInBounds := thatIdx < len(other.results)
	for thisInBounds && thatInBounds {
		cmp := strings.Compare(rs.results[thisIdx].ID, other.results[thatIdx].ID)
		if cmp == 0 {
			mergeTo(&rs.results[thisIdx], &other.results[thatIdx])
			newResults = append(newResults, rs.results[thisIdx])
			thatIdx++
			thisIdx++
		} else if cmp > 0 {
			thatIdx++
		} else {
			thisIdx++
		}
		thisInBounds = thisIdx < len(rs.results)
		thatInBounds = thatIdx < len(other.results)
	}

	return resultSet{
		results: newResults,
		order:   order,
	}
}

func (rs resultSet) union(other resultSet) resultSet {
	order := rs.order
	if other.order != nil {
		order = other.order
	}

	newResults := make([]search.Result, 0, len(rs.results))
	thatIdx := 0
	thisIdx := 0
	thisInBounds := thisIdx < len(rs.results)
	thatInBounds := thatIdx < len(other.results)
	for thisInBounds || thatInBounds {
		var cmp int
		if thisInBounds && thatInBounds {
			cmp = strings.Compare(rs.results[thisIdx].ID, other.results[thatIdx].ID)
		} else if thatInBounds {
			cmp = 1
		} else {
			cmp = -1
		}
		if cmp == 0 {
			mergeTo(&rs.results[thisIdx], &other.results[thatIdx])
			newResults = append(newResults, rs.results[thisIdx])
			thatIdx++
			thisIdx++
		} else if cmp > 0 {
			newResults = append(newResults, other.results[thatIdx])
			thatIdx++
		} else {
			newResults = append(newResults, rs.results[thisIdx])
			thisIdx++
		}
		thisInBounds = thisIdx < len(rs.results)
		thatInBounds = thatIdx < len(other.results)
	}

	return resultSet{
		results: newResults,
		order:   order,
	}
}

func (rs resultSet) subtract(other resultSet) resultSet {
	order := rs.order
	if other.order != nil {
		order = other.order
	}

	newResults := make([]search.Result, 0, len(rs.results))
	thatIdx := 0
	thisIdx := 0
	thisInBounds := thisIdx < len(rs.results)
	thatInBounds := thatIdx < len(other.results)
	for thisInBounds {
		cmp := -1
		if thatInBounds {
			cmp = strings.Compare(rs.results[thisIdx].ID, other.results[thatIdx].ID)
		}

		if cmp == 0 {
			thatIdx++
			thisIdx++
		} else if cmp > 0 {
			thatIdx++
		} else {
			newResults = append(newResults, rs.results[thisIdx])
			thisIdx++
		}
		thisInBounds = thisIdx < len(rs.results)
		thatInBounds = thatIdx < len(other.results)
	}

	return resultSet{
		results: newResults,
		order:   order,
	}
}

func (rs *resultSet) asResultSlice() []search.Result {
	ret := rs.results
	if rs.order != nil {
		ret = make([]search.Result, len(rs.results))
		for _, res := range rs.results {
			ret[rs.order[res.ID]] = res
		}
	}
	return ret
}

// Merge any retrieved fields from one result to another. Helpful if fields have been requested in the results.
func mergeTo(to, from *search.Result) {
	if to.Matches == nil && from.Matches != nil {
		to.Matches = from.Matches
	} else {
		for k, vs := range from.Matches {
			if _, toHas := to.Matches[k]; !toHas {
				to.Matches[k] = vs
			}
		}
	}

	if to.Fields == nil && from.Fields != nil {
		to.Fields = from.Fields
	} else {
		for k, vs := range from.Fields {
			if _, toHas := to.Fields[k]; !toHas {
				to.Fields[k] = vs
			}
		}
	}
}
