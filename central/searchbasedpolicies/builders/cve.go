package builders

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
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
		err = fmt.Errorf("regex '%s' invalid: %s", cve, err)
		return
	}

	cveSearchField, err := getSearchField(search.CVE, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", c.Name(), err)
		return
	}
	cveLinkSearchField, err := getSearchField(search.CVELink, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", c.Name(), err)
		return
	}

	q = search.NewQueryBuilder().AddLinkedFieldsHighlighted(
		[]search.FieldLabel{search.CVE, search.CVELink},
		[]string{search.RegexQueryString(cve), search.WildcardString}).
		ProtoQuery()
	v = func(result search.Result, _ searchbasedpolicies.ProcessIndicatorGetter) searchbasedpolicies.Violations {
		cveMatches := result.Matches[cveSearchField.GetFieldPath()]
		cveLinkMatches := result.Matches[cveLinkSearchField.GetFieldPath()]
		if len(cveMatches) != len(cveLinkMatches) {
			logger.Errorf("Got different number of matches for CVEs and links: %+v %+v", cveMatches, cveLinkMatches)
		}
		if len(cveMatches) == 0 {
			return searchbasedpolicies.Violations{}
		}
		violations := make([]*storage.Alert_Violation, 0, len(cveMatches))
		for i, cveMatch := range cveMatches {
			var link string
			if len(cveLinkMatches) > i {
				link = cveLinkMatches[i]
			}
			violations = append(violations, &storage.Alert_Violation{
				Message: fmt.Sprintf("CVE %s matched regex '%s'", cveMatch, cve),
				Link:    link,
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
