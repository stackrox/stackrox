package builders

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

type regexFieldQueryBuilder struct {
	fieldLabel         search.FieldLabel
	fieldHumanName     string
	retrieveFieldValue func(*v1.PolicyFields) string
	allowSubstrings    bool
}

func (c *regexFieldQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for %s", c.fieldHumanName)
}

func (c *regexFieldQueryBuilder) Query(fields *v1.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	policyVal := c.retrieveFieldValue(fields)
	if policyVal == "" {
		return
	}

	searchField, err := getSearchField(c.fieldLabel, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", c.Name(), err)
		return
	}

	actualPolicyVal := policyVal
	if c.allowSubstrings {
		actualPolicyVal = fmt.Sprintf(".*%s.*", policyVal)
	}

	// Make sure the regex compiles (Bleve will just fail silently.)
	_, err = regexp.Compile(actualPolicyVal)
	if err != nil {
		err = fmt.Errorf("'%s' is an invalid regex: %s", actualPolicyVal, err)
		return
	}

	q = search.NewQueryBuilder().AddRegexesHighlighted(c.fieldLabel, actualPolicyVal).ProtoQuery()
	v = violationPrinterForField(searchField.GetFieldPath(), func(match string) string {
		return fmt.Sprintf("%s '%s' matched %s", c.fieldHumanName, match, policyVal)
	})

	return
}

// NewRegexQueryBuilder returns a query builder that constructs a regex query for the given field.
func NewRegexQueryBuilder(fieldLabel search.FieldLabel, fieldHumanName string, retrieveFieldValue func(*v1.PolicyFields) string) searchbasedpolicies.PolicyQueryBuilder {
	return newRegexQueryBuilder(fieldLabel, fieldHumanName, retrieveFieldValue, false)
}

// NewRegexQueryBuilderWithSubstrings returns a query builder that constructs a regex query for the given field which also allows substrings.
func NewRegexQueryBuilderWithSubstrings(fieldLabel search.FieldLabel, fieldHumanName string, retrieveFieldValue func(*v1.PolicyFields) string) searchbasedpolicies.PolicyQueryBuilder {
	return newRegexQueryBuilder(fieldLabel, fieldHumanName, retrieveFieldValue, true)
}

func newRegexQueryBuilder(fieldLabel search.FieldLabel, fieldHumanName string, retrieveFieldValue func(*v1.PolicyFields) string, allowSubstrings bool) searchbasedpolicies.PolicyQueryBuilder {
	return &regexFieldQueryBuilder{
		fieldLabel:         fieldLabel,
		fieldHumanName:     fieldHumanName,
		retrieveFieldValue: retrieveFieldValue,
		allowSubstrings:    allowSubstrings,
	}
}
