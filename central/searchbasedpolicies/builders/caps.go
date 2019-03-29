package builders

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// addCapQueryBuilder builds queries for add and drop capability queries.
type addCapQueryBuilder struct {
}

func (c addCapQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	addCaps := fields.GetAddCapabilities()
	if len(addCaps) == 0 {
		return
	}

	searchField, err := getSearchField(search.AddCapabilities, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", c.Name())
		return
	}

	q = search.NewQueryBuilder().AddStringsHighlighted(search.AddCapabilities, addCaps...).ProtoQuery()
	v = violationPrinterForField(searchField.GetFieldPath(), func(match string) string {
		return fmt.Sprintf("%s was in the ADD CAPABILITIES list", match)
	})
	return
}

func (c addCapQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for add capabilities")
}

// addCapQueryBuilder builds queries for add and drop capability queries.
type dropCapQueryBuilder struct {
}

func (c dropCapQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	dropCaps := fields.GetDropCapabilities()
	if len(dropCaps) == 0 {
		return
	}

	searchField, err := getSearchField(search.DropCapabilities, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", c.Name())
		return
	}

	q = search.NewQueryBuilder().AddStringsHighlighted(search.DropCapabilities, dropCaps...).ProtoQuery()
	v = violationPrinterForField(searchField.GetFieldPath(), func(match string) string {
		return fmt.Sprintf("%s was in the DROP CAPABILITIES list", match)
	})
	return
}

// Name implements the PolicyQueryBuilder interface.
func (c dropCapQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for drop capabilities")
}

// NewAddCapQueryBuilder returns a ready-to-use query builder for the add capabilities field in policies.
func NewAddCapQueryBuilder() searchbasedpolicies.PolicyQueryBuilder {
	return addCapQueryBuilder{}
}

// NewDropCapQueryBuilder returns a ready-to-use query builder for the drop capabilities field in policies.
func NewDropCapQueryBuilder() searchbasedpolicies.PolicyQueryBuilder {
	return dropCapQueryBuilder{}
}
