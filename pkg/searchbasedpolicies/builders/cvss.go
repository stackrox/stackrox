package builders

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

// CVSSQueryBuilder builds queries for the CVSS field in policies.
type CVSSQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (c CVSSQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	cvss := fields.GetCvss()
	fixedBy := fields.GetFixedBy()

	if cvss == nil && fixedBy == "" {
		return
	}

	cvssSearchField, err := getSearchField(search.CVSS, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", c.Name())
		return
	}
	cveSearchField, err := getSearchField(search.CVE, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", c.Name())
		return
	}

	cveFixedByField, err := getSearchField(search.FixedBy, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", c.Name())
		return
	}

	linkedFields := []search.FieldLabel{search.CVSS, search.CVE, search.CVESuppressed}
	linkedValues := []interface{}{search.NumericQueryString(cvss.GetOp(), cvss.GetValue()), search.WildcardString, false}
	if fixedBy != "" {
		linkedFields = append(linkedFields, search.FixedBy)
		linkedValues = append(linkedValues, search.RegexQueryString(fixedBy))
	}

	q = search.NewQueryBuilder().AddGenericTypeLinkedFieldsHighligted(linkedFields, linkedValues).ProtoQuery()
	v = func(_ context.Context, result search.Result) searchbasedpolicies.Violations {
		cvssMatches := result.Matches[cvssSearchField.GetFieldPath()]
		if len(cvssMatches) == 0 {
			cvssMatches = result.Matches[swapFieldPaths(cvssSearchField.GetFieldPath())]
		}
		cveMatches := result.Matches[cveSearchField.GetFieldPath()]
		if len(cveMatches) == 0 {
			cveMatches = result.Matches[swapFieldPaths(cveSearchField.GetFieldPath())]
		}
		fixedByMatches := result.Matches[cveFixedByField.GetFieldPath()]
		if len(fixedByMatches) == 0 {
			fixedByMatches = result.Matches[swapFieldPaths(cveFixedByField.GetFieldPath())]
		}
		if len(cvssMatches) != len(cveMatches) {
			log.Errorf("Got different number of matches for CVSS and CVEs: %+v %+v", cvssMatches, cveMatches)
		}
		if len(cvssMatches) == 0 {
			return searchbasedpolicies.Violations{}
		}
		violations := make([]*storage.Alert_Violation, 0, len(cvssMatches))
		for i, cvssMatch := range cvssMatches {
			if i >= len(cveMatches) {
				break
			}
			cve := fmt.Sprintf(" (cve: %s)", cveMatches[i])
			var msg string
			if len(fixedByMatches) > i {
				msg = fmt.Sprintf("Found a CVSS score of %s (%s %.1f)%s that is fixable", cvssMatch, readableOp(cvss.GetOp()), cvss.GetValue(), cve)
			} else {
				msg = fmt.Sprintf("Found a CVSS score of %s (%s %.1f)%s", cvssMatch, readableOp(cvss.GetOp()), cvss.GetValue(), cve)
			}
			violations = append(violations, &storage.Alert_Violation{
				Message: msg,
			})
		}
		return searchbasedpolicies.Violations{
			AlertViolations: violations,
		}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (c CVSSQueryBuilder) Name() string {
	return "Query builder for CVSS"
}
