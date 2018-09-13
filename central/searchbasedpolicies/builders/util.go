package builders

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	logger = logging.LoggerForModule()
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
