package builders

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// DisallowedMapValueQueryBuilder builds queries to check for the existence of a map value.
type DisallowedMapValueQueryBuilder struct {
	GetKeyValuePolicy func(*storage.PolicyFields) *storage.KeyValuePolicy
	FieldName         string
	FieldLabel        search.FieldLabel
}

// Query implements the PolicyQueryBuilder interface.
func (r DisallowedMapValueQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	keyValuePolicy := r.GetKeyValuePolicy(fields)
	if keyValuePolicy.GetKey() == "" {
		if keyValuePolicy.GetValue() != "" {
			err = fmt.Errorf("key value policy for %s had no key, only a value: %s", r.FieldName, keyValuePolicy.GetValue())
			return
		}
		return
	}

	_, err = getSearchFieldNotStored(r.FieldLabel, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", r.Name(), err)
		return
	}

	var valueQuery string
	if keyValuePolicy.GetValue() == "" {
		valueQuery = search.WildcardString
	} else {
		valueQuery = search.RegexQueryString(keyValuePolicy.GetValue())
	}
	q = search.NewQueryBuilder().AddMapQuery(r.FieldLabel, keyValuePolicy.GetKey(), valueQuery).ProtoQuery()

	v = func(result search.Result, _ searchbasedpolicies.ProcessIndicatorGetter) []*storage.Alert_Violation {
		return []*storage.Alert_Violation{{Message: fmt.Sprintf("Disallowed %s found (%s)", r.FieldName, printKeyValuePolicy(keyValuePolicy))}}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (r DisallowedMapValueQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for disallowed map value %s", r.FieldName)
}
