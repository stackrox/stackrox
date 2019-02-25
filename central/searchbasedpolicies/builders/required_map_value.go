package builders

import (
	"fmt"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// RequiredMapValueQueryBuilder builds queries to check for the (absence of) a required map value.
type RequiredMapValueQueryBuilder struct {
	GetKeyValuePolicy func(*storage.PolicyFields) *storage.KeyValuePolicy
	FieldName         string
	FieldLabel        search.FieldLabel
}

// Query implements the PolicyQueryBuilder interface.
func (r RequiredMapValueQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
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
		valueQuery = search.NullQueryString()
	} else {
		valueQuery = search.NegateQueryString(search.RegexQueryString(keyValuePolicy.GetValue()))
	}
	queryIfKeyExist := search.NewQueryBuilder().AddMapQuery(r.FieldLabel, search.ExactMatchString(keyValuePolicy.GetKey()), valueQuery).ProtoQuery()
	queryIfKeyDoesNotExist := search.NewQueryBuilder().AddMapQuery(r.FieldLabel, search.NegateQueryString(search.ExactMatchString(keyValuePolicy.GetKey())), "").ProtoQuery()

	q = &v1.Query{
		Query: &v1.Query_Disjunction{
			Disjunction: &v1.DisjunctionQuery{
				Queries: []*v1.Query{queryIfKeyExist, queryIfKeyDoesNotExist},
			},
		},
	}

	v = func(result search.Result, _ searchbasedpolicies.ProcessIndicatorGetter) searchbasedpolicies.Violations {
		return searchbasedpolicies.Violations{
			AlertViolations: []*storage.Alert_Violation{
				{Message: fmt.Sprintf("Required %s not found (%s)", r.FieldName, printKeyValuePolicy(keyValuePolicy))},
			},
		}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (r RequiredMapValueQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for required %s", r.FieldName)
}
