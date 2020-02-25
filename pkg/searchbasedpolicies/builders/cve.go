package builders

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

// CVEQueryBuilder builds queries for the CVE field in policies.
type CVEQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (c CVEQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	cve := fields.GetCve()
	if cve == "" {
		return
	}

	_, err = regexp.Compile(cve)
	if err != nil {
		err = errors.Wrapf(err, "regex '%s' invalid", cve)
		return
	}

	cveSearchField, err := getSearchField(search.CVE, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", c.Name())
		return
	}

	// Exclude suppressed cves for violation computation
	fieldLabels := []search.FieldLabel{search.CVE, search.CVESuppressed}
	queryStrings := []interface{}{search.RegexQueryString(cve), false}

	q = search.NewQueryBuilder().AddGenericTypeLinkedFieldsHighligted(fieldLabels, queryStrings).ProtoQuery()
	v = func(_ context.Context, result search.Result) searchbasedpolicies.Violations {
		cveMatches := result.Matches[cveSearchField.GetFieldPath()]
		if len(cveMatches) == 0 {
			return searchbasedpolicies.Violations{}
		}
		violations := make([]*storage.Alert_Violation, 0, len(cveMatches))
		for _, cveMatch := range cveMatches {
			violations = append(violations, &storage.Alert_Violation{
				Message: fmt.Sprintf("CVE %s matched regex '%s'", cveMatch, cve),
			})
		}
		return searchbasedpolicies.Violations{
			AlertViolations: violations,
		}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (CVEQueryBuilder) Name() string {
	return "Query builder for CVEs"
}
