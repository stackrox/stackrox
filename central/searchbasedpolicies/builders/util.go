package builders

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	logger = logging.LoggerForModule()

	opToHumanReadable = map[v1.Comparator]string{
		v1.Comparator_LESS_THAN:              "less than",
		v1.Comparator_LESS_THAN_OR_EQUALS:    "less than or equal to",
		v1.Comparator_EQUALS:                 "equal to",
		v1.Comparator_GREATER_THAN_OR_EQUALS: "greater than or equal to",
		v1.Comparator_GREATER_THAN:           "greater than",
	}
)

func getSearchField(fieldLabel search.FieldLabel, optionsMap map[search.FieldLabel]*v1.SearchField) (*v1.SearchField, error) {
	searchField, exists := optionsMap[fieldLabel]
	if !exists {
		return nil, fmt.Errorf("couldn't construct query: field %s not found in options map", fieldLabel)
	}
	if !searchField.GetStore() {
		return nil, fmt.Errorf("field %s is required for search, but not stored", fieldLabel)
	}
	return searchField, nil
}

func violationPrinterForField(fieldPath string, matchToMessage func(match string) string) searchbasedpolicies.ViolationPrinter {
	return func(result search.Result) []*v1.Alert_Violation {
		matches := result.Matches[fieldPath]
		if len(matches) == 0 {
			return nil
		}
		violations := make([]*v1.Alert_Violation, 0, len(matches))
		for _, match := range matches {
			if message := matchToMessage(match); message != "" {
				violations = append(violations, &v1.Alert_Violation{Message: matchToMessage(match)})
			}
		}
		return violations

	}
}

func printKeyValuePolicy(kvp *v1.KeyValuePolicy) string {
	sb := strings.Builder{}
	if kvp.GetKey() != "" {
		sb.WriteString(fmt.Sprintf("key = '%s'", kvp.GetKey()))
		if kvp.GetValue() != "" {
			sb.WriteString(", ")
		}
	}
	if kvp.GetValue() != "" {
		sb.WriteString(fmt.Sprintf("value = '%s'", kvp.GetValue()))
	}
	return sb.String()
}

func concatenatingPrinter(printers []searchbasedpolicies.ViolationPrinter) searchbasedpolicies.ViolationPrinter {
	return func(result search.Result) (violations []*v1.Alert_Violation) {
		for _, p := range printers {
			violations = append(violations, p(result)...)
		}
		return
	}
}

func presentQueriesAndPrinters(qbs []searchbasedpolicies.PolicyQueryBuilder, fields *v1.PolicyFields,
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

func readableOp(op v1.Comparator) string {
	return opToHumanReadable[op]
}
