package pgsearch

import (
	"fmt"
	"math"
	"strconv"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

type QueryEntry struct {
	Query  string
	Values []interface{}
}

func NewFalseQuery() *QueryEntry {
	return &QueryEntry{
		Query: "false",
	}
}

func NewTrueQuery() *QueryEntry {
	return &QueryEntry{
		Query: "true",
	}
}

func generateShortestElemPath(table string, elems []searchPkg.PathElem) string {
	if len(elems) == 1 {
		return fmt.Sprintf("%s.value", table)
	}
	if lastElem := elems[len(elems)-1]; lastElem.Slice {
		return ""
	}
	var path string
	for i := len(elems) - 2; i > -1; i-- {
		elem := elems[i]
		if elem.Slice {
			path = elems[i].Name + path
			return path
		}
		if i == 0 {
			path = fmt.Sprintf("%s.value->'%s'", table, elems[i].Name) + path
			return path
		}
		path = fmt.Sprintf("->'%s'", elems[i].Name) + path
	}
	return path
}

func MatchFieldQuery(table string, query *v1.MatchFieldQuery, optionsMap searchPkg.OptionsMap) (*QueryEntry, error) {
	// Need to find base value
	field, ok := optionsMap.Get(query.GetField())
	if !ok {
		log.Infof("Options Map for %s does not have field: %v", table, query.GetField())
		return nil, nil
	}
	return matchFieldQuery(table, field, query.Value)
}

func getValueFromField(val interface{}) string {
	switch val := val.(type) {
	case string:
		return val
	case float64:
		i, f := math.Modf(val)
		// If it's an int, return just the int portion.
		if math.Abs(f) < 1e-3 {
			return fmt.Sprintf("%d", int(i))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		return strconv.FormatBool(val)
	default:
		log.Errorf("Unknown type field from index: %T", val)
	}
	return ""
}

//
//func getSortOrderAndSearchAfter(pagination *v1.QueryPagination, optionsMap searchPkg.OptionsMap) (search.SortOrder, []string, error) {
//	if len(pagination.GetSortOptions()) == 0 {
//		return nil, nil, nil
//	}
//
//	sortOrder := make([]search.SearchSort, 0, len(pagination.GetSortOptions()))
//
//	var searchAfter []string
//	searchAfterHasDocID := false
//	allowSearchAfter := true
//
//	for _, so := range pagination.GetSortOptions() {
//		var sortField search.SearchSort
//
//		if so.GetField() == searchPkg.DocID.String() {
//			sortField = &search.SortDocID{
//				Desc: so.GetReversed(),
//			}
//		} else {
//			sf, ok := optionsMap.Get(so.GetField())
//			if !ok {
//				return nil, nil, errors.Errorf("option %q is not a valid search option", so.GetField())
//			}
//			sortField = &search.SortField{
//				Field:   sf.GetFieldPath(),
//				Desc:    so.GetReversed(),
//				Type:    search.SortFieldAuto,
//				Missing: search.SortFieldMissingLast,
//			}
//		}
//
//		sortOrder = append(sortOrder, sortField)
//
//		if saOpt, ok := so.GetSearchAfterOpt().(*v1.QuerySortOption_SearchAfter); ok {
//			if !allowSearchAfter {
//				return nil, nil, errors.New("invalid SearchAfter state: SearchAfter values must start from the beginning of SortOptions and must follow without gaps")
//			}
//			if so.GetField() == searchPkg.DocID.String() {
//				searchAfterHasDocID = true
//			}
//			searchAfter = append(searchAfter, saOpt.SearchAfter)
//		} else {
//			allowSearchAfter = false
//		}
//	}
//
//	if len(searchAfter) > 0 && !searchAfterHasDocID {
//		// This checks that SearchAfter will have effect when used or returns an error.
//		// It appears that Bleve does not have validations for bleve.SearchRequest.SearchAfter. This closes the gap.
//		// See https://github.com/blevesearch/bleve/pull/1182#issuecomment-499216058
//		return nil, nil, utils.Should(errors.New("total ordering not guaranteed: SortOrder must contain DocID and SearchAfter value for it to ensure there are no ties, otherwise SearchAfter will not produce correct results"))
//	}
//
//	return sortOrder, searchAfter, nil
//}



