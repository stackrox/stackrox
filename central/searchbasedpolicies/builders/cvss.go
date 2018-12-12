package builders

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// CVSSQueryBuilder builds queries for the CVSS field in policies.
type CVSSQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (c CVSSQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	cvss := fields.GetCvss()
	if cvss == nil {
		return
	}

	cvssSearchField, err := getSearchField(search.CVSS, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", c.Name(), err)
		return
	}
	cveSearchField, err := getSearchField(search.CVE, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", c.Name(), err)
		return
	}

	q = search.NewQueryBuilder().AddLinkedFieldsHighlighted(
		[]search.FieldLabel{search.CVSS, search.CVE},
		[]string{search.NumericQueryString(cvss.GetOp(), cvss.GetValue()), search.WildcardString}).
		ProtoQuery()
	v = func(result search.Result, _ searchbasedpolicies.ProcessIndicatorGetter) []*v1.Alert_Violation {
		cvssMatches := result.Matches[cvssSearchField.GetFieldPath()]
		cveMatches := result.Matches[cveSearchField.GetFieldPath()]
		if len(cvssMatches) != len(cveMatches) {
			logger.Errorf("Got different number of matches for CVSS and CVEs: %+v %+v", cvssMatches, cveMatches)
		}
		if len(cvssMatches) == 0 {
			return nil
		}
		violations := make([]*v1.Alert_Violation, 0, len(cvssMatches))
		for i, cvssMatch := range cvssMatches {
			if i >= len(cveMatches) {
				break
			}
			cve := fmt.Sprintf(" (cve: %s)", cveMatches[i])
			violations = append(violations, &v1.Alert_Violation{
				Message: fmt.Sprintf("Found a CVSS score of %s (%s %.1f)%s", cvssMatch, readableOp(cvss.GetOp()), cvss.GetValue(), cve),
			})
		}
		return violations
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (c CVSSQueryBuilder) Name() string {
	return "Query builder for CVSS"
}
