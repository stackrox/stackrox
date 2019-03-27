package builders

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type resourcePolicyBuilder struct {
	extractFieldValue func(fields *storage.PolicyFields) *storage.NumericalPolicy
	fieldLabel        search.FieldLabel
	fieldHumanName    string
}

func (r *resourcePolicyBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	numericalPolicy := r.extractFieldValue(fields)
	if numericalPolicy == nil {
		return
	}
	searchField, err := getSearchField(r.fieldLabel, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", r.Name(), err)
		return
	}

	q = search.NewQueryBuilder().AddNumericFieldHighlighted(r.fieldLabel, numericalPolicy.GetOp(), numericalPolicy.GetValue()).ProtoQuery()

	v = violationPrinterForField(searchField.GetFieldPath(), func(match string) string {
		return fmt.Sprintf("The %s of %s is %s the threshold of %.2f", r.fieldHumanName, match, readableOp(numericalPolicy.GetOp()), numericalPolicy.GetValue())
	})
	return
}

func (r *resourcePolicyBuilder) Name() string {
	return fmt.Sprintf("query builder for resource policy: %s", r.fieldHumanName)
}

// NewResourcePolicyBuilder returns a resource policy builder with the specified parameters.
func NewResourcePolicyBuilder(extractFieldValue func(fields *storage.PolicyFields) *storage.NumericalPolicy, fieldLabel search.FieldLabel, fieldHumanName string) searchbasedpolicies.PolicyQueryBuilder {
	return &resourcePolicyBuilder{
		extractFieldValue: extractFieldValue,
		fieldHumanName:    fieldHumanName,
		fieldLabel:        fieldLabel,
	}
}
