package builders

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

var (
	log = logging.LoggerForModule()

	opToHumanReadable = map[storage.Comparator]string{
		storage.Comparator_LESS_THAN:              "less than",
		storage.Comparator_LESS_THAN_OR_EQUALS:    "less than or equal to",
		storage.Comparator_EQUALS:                 "equal to",
		storage.Comparator_GREATER_THAN_OR_EQUALS: "greater than or equal to",
		storage.Comparator_GREATER_THAN:           "greater than",
	}
)

func regexOrWildcard(valueInPolicy string) string {
	if valueInPolicy == "" {
		return search.WildcardString
	}
	return search.RegexQueryString(valueInPolicy)
}

func getSearchField(fieldLabel search.FieldLabel, optionsMap map[search.FieldLabel]*v1.SearchField) (*v1.SearchField, error) {
	searchField, err := getSearchFieldNotStored(fieldLabel, optionsMap)
	if err != nil {
		return nil, err
	}
	if !searchField.GetStore() {
		return nil, fmt.Errorf("field %s is required for search, but not stored", fieldLabel)
	}
	return searchField, nil
}

func getSearchFieldNotStored(fieldLabel search.FieldLabel, optionsMap map[search.FieldLabel]*v1.SearchField) (*v1.SearchField, error) {
	searchField, exists := optionsMap[fieldLabel]
	if !exists {
		return nil, fmt.Errorf("couldn't construct query: field %s not found in options map", fieldLabel)
	}
	return searchField, nil
}

func violationPrinterForField(fieldPath string, matchToMessage func(match string) string) searchbasedpolicies.ViolationPrinter {
	return func(_ context.Context, result search.Result) searchbasedpolicies.Violations {
		matches := result.Matches[fieldPath]
		if len(matches) == 0 {
			return searchbasedpolicies.Violations{}
		}
		violations := make([]*storage.Alert_Violation, 0, len(matches))
		for _, match := range matches {
			if message := matchToMessage(match); message != "" {
				violations = append(violations, &storage.Alert_Violation{Message: matchToMessage(match)})
			}
		}
		return searchbasedpolicies.Violations{
			AlertViolations: violations,
		}
	}
}

func printKeyValuePolicy(kvp *storage.KeyValuePolicy) string {
	sb := strings.Builder{}
	if kvp.GetKey() != "" {
		_, _ = sb.WriteString(fmt.Sprintf("key = '%s'", kvp.GetKey()))
		if kvp.GetValue() != "" || kvp.GetEnvVarSource() != storage.ContainerConfig_EnvironmentConfig_UNSET {
			_, _ = sb.WriteString(", ")
		}
	}
	if kvp.GetValue() != "" {
		_, _ = sb.WriteString(fmt.Sprintf("value = '%s'", kvp.GetValue()))
		if kvp.GetEnvVarSource() != storage.ContainerConfig_EnvironmentConfig_UNSET {
			_, _ = sb.WriteString(", ")
		}
	}
	if kvp.GetEnvVarSource() != storage.ContainerConfig_EnvironmentConfig_UNSET {
		_, _ = sb.WriteString(fmt.Sprintf("value from = '%s'", kvp.GetEnvVarSource()))
	}
	return sb.String()
}

func concatenatingPrinter(printers []searchbasedpolicies.ViolationPrinter) searchbasedpolicies.ViolationPrinter {
	return func(ctx context.Context, result search.Result) searchbasedpolicies.Violations {
		concatenated := searchbasedpolicies.Violations{}
		for _, p := range printers {
			violation := p(ctx, result)
			// Only one process violation will be present.
			if violation.ProcessViolation != nil {
				concatenated.ProcessViolation = violation.ProcessViolation
			}
			if len(violation.AlertViolations) > 0 {
				concatenated.AlertViolations = append(concatenated.AlertViolations, violation.AlertViolations...)
			}
		}
		return concatenated
	}
}

func presentQueriesAndPrinters(qbs []searchbasedpolicies.PolicyQueryBuilder, fields *storage.PolicyFields,
	optionsMap map[search.FieldLabel]*v1.SearchField) (queries []*v1.Query, printers []searchbasedpolicies.ViolationPrinter, err error) {
	for _, qb := range qbs {
		var q *v1.Query
		var printer searchbasedpolicies.ViolationPrinter
		q, printer, err = qb.Query(fields, optionsMap)
		if err != nil {
			return
		}
		if q == nil {
			continue
		}
		if printer == nil {
			err = fmt.Errorf("query builder %+v (%s) returned non-nil query but nil printer", qb, qb.Name())
			return
		}
		queries = append(queries, q)
		printers = append(printers, printer)
	}
	return
}

func readableOp(op storage.Comparator) string {
	return opToHumanReadable[op]
}

// We need to try alternate field paths to get the violation message so that if search is running on the top level
// component and CVE object we can still pull the matches.
var dackboxSwappedFieldPaths = map[string]string{
	"image.scan.components.name":                        "image_component.name",
	"image.scan.components.version":                     "image_component.version",
	"image.scan.components.vulns.cve":                   "c_v_e.id",
	"image.scan.components.vulns.cvss":                  "c_v_e.cvss",
	"image.scan.components.vulns.set_fixed_by.fixed_by": "component_c_v_e_edge.has_fixed_by.fixed_by",
}

func swapFieldPaths(fieldPath string) string {
	if np, exists := dackboxSwappedFieldPaths[fieldPath]; exists {
		return np
	}
	return fieldPath
}
