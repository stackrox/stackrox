package builders

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// processNameQueryBuilder builds queries for process name field.
type processNameQueryBuilder struct {
}

func (p processNameQueryBuilder) Query(fields *v1.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	processPolicy := fields.GetProcessPolicy()
	if processPolicy == nil {
		return
	}
	processName := processPolicy.Name
	if len(processName) == 0 {
		return
	}

	searchField, err := getSearchField(search.ProcessName, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", processName, err)
		return
	}

	q = search.NewQueryBuilder().AddStringsHighlighted(search.ProcessName, processName).ProtoQuery()
	v = violationPrinterForField(searchField.GetFieldPath(), func(match string) string {
		return fmt.Sprintf("%s was in the process name", match)
	})
	return
}

func (p processNameQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for process name")
}

// processArgsQueryBuilder builds queries for process args queries.
type processArgsQueryBuilder struct {
}

func (p processArgsQueryBuilder) Query(fields *v1.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	processPolicy := fields.GetProcessPolicy()
	if processPolicy == nil {
		return
	}
	processArgs := processPolicy.Args
	if len(processArgs) == 0 {
		return
	}

	searchField, err := getSearchField(search.ProcessArguments, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", processArgs, err)
		return
	}

	q = search.NewQueryBuilder().AddStringsHighlighted(search.ProcessArguments, processArgs).ProtoQuery()
	v = violationPrinterForField(searchField.GetFieldPath(), func(match string) string {
		return fmt.Sprintf("%s was in the process arguments", match)
	})
	return
}

// Name implements the PolicyQueryBuilder interface.
func (p processArgsQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for process arguments")
}

// NewProcessNameQueryBuilder returns a ready-to-use query builder for the process name field in policies.
func NewProcessNameQueryBuilder() searchbasedpolicies.PolicyQueryBuilder {
	return processNameQueryBuilder{}
}

// NewProcessArgsQueryBuilder returns a ready-to-use query builder for the process args field in policies.
func NewProcessArgsQueryBuilder() searchbasedpolicies.PolicyQueryBuilder {
	return processArgsQueryBuilder{}
}
