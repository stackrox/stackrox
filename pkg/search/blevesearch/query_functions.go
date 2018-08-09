package blevesearch

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/generated/api/v1"
)

var datatypeToQueryFunc = map[v1.SearchDataType]func(string, []string) (query.Query, error){
	v1.SearchDataType_SEARCH_STRING:      newStringQuery,
	v1.SearchDataType_SEARCH_BOOL:        newBoolQuery,
	v1.SearchDataType_SEARCH_NUMERIC:     newNumericQuery,
	v1.SearchDataType_SEARCH_SEVERITY:    newSeverityQuery,
	v1.SearchDataType_SEARCH_ENFORCEMENT: newEnforcementQuery,
}

func newStringQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, val := range values {
		d.AddQuery(NewMatchPhrasePrefixQuery(field, val))
	}
	return d, nil
}

func newBoolQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, val := range values {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		q := bleve.NewBoolFieldQuery(b)
		q.FieldVal = field
		d.AddQuery(q)
	}
	return d, nil
}

func boolPtr(b bool) *bool {
	return &b
}

func parseNumericStringToPtr(s string) (*float64, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func parseNumericValue(num string) (min *float64, max *float64, inclusive *bool, err error) {
	if strings.HasPrefix(num, "<=") {
		inclusive = boolPtr(true)
		max, err = parseNumericStringToPtr(strings.TrimPrefix(num, "<="))
	} else if strings.HasPrefix(num, "<") {
		inclusive = boolPtr(false)
		max, err = parseNumericStringToPtr(strings.TrimPrefix(num, "<"))
	} else if strings.HasPrefix(num, ">=") {
		inclusive = boolPtr(true)
		min, err = parseNumericStringToPtr(strings.TrimPrefix(num, ">="))
	} else if strings.HasPrefix(num, ">") {
		inclusive = boolPtr(false)
		min, err = parseNumericStringToPtr(strings.TrimPrefix(num, ">"))
	} else {
		inclusive = boolPtr(true)
		min, err = parseNumericStringToPtr(num)
		max = min
	}
	return
}

func newNumericQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, val := range values {
		min, max, inclusive, err := parseNumericValue(val)
		if err != nil {
			return nil, err
		}
		q := bleve.NewNumericRangeInclusiveQuery(min, max, inclusive, inclusive)
		q.FieldVal = field
		d.AddQuery(q)
	}
	return d, nil
}

func stringToSeverity(s string) (v1.Severity, error) {
	s = strings.ToLower(s)
	if strings.HasPrefix(s, "l") {
		return v1.Severity_LOW_SEVERITY, nil
	}
	if strings.HasPrefix(s, "m") {
		return v1.Severity_MEDIUM_SEVERITY, nil
	}
	if strings.HasPrefix(s, "h") {
		return v1.Severity_HIGH_SEVERITY, nil
	}
	if strings.HasPrefix(s, "c") {
		return v1.Severity_CRITICAL_SEVERITY, nil
	}
	return v1.Severity_UNSET_SEVERITY, fmt.Errorf("Could not parse severity '%s'. Valid options are low, medium, high, critical", s)
}

func newExactNumericMatch(field string, f float64) query.Query {
	t := true
	q := bleve.NewNumericRangeInclusiveQuery(&f, &f, &t, &t)
	q.FieldVal = field
	return q
}

func newSeverityQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, v := range values {
		sev, err := stringToSeverity(v)
		if err != nil {
			return nil, err
		}
		d.AddQuery(newExactNumericMatch(field, float64(sev)))
	}
	return d, nil
}

func stringToEnforcement(s string) (v1.EnforcementAction, error) {
	s = strings.ToLower(s)
	if strings.Contains(s, "scale") {
		return v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, nil
	}
	if strings.Contains(s, "node") {
		return v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT, nil
	}
	if strings.Contains(s, "none") {
		return v1.EnforcementAction_UNSET_ENFORCEMENT, nil
	}
	return v1.EnforcementAction_UNSET_ENFORCEMENT, fmt.Errorf("Could not parse enforcement '%s'. Valid options are node, and scale", s)
}

func newEnforcementQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, v := range values {
		en, err := stringToEnforcement(v)
		if err != nil {
			return nil, err
		}
		d.AddQuery(newExactNumericMatch(field, float64(en)))
	}
	return d, nil
}
