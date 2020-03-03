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

func mapKeyValueQuery(optionsMap map[search.FieldLabel]*v1.SearchField, keyValuePolicy *storage.KeyValuePolicy, fieldLabel search.FieldLabel, fieldName string, name string, keyWrapper func(string) string) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	if keyValuePolicy.GetKey() == "" {
		if keyValuePolicy.GetValue() != "" {
			err = fmt.Errorf("key value policy for %s had no key, only a value: %s", fieldName, keyValuePolicy.GetValue())
			return
		}
		return
	}

	_, err = getSearchFieldNotStored(fieldLabel, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", name)
		return
	}

	var valueQuery string
	if keyValuePolicy.GetValue() == "" {
		valueQuery = search.NullQueryString()
	} else {
		valueQuery = search.NegateQueryString(search.RegexQueryString(keyValuePolicy.GetValue()))
	}
	queryIfKeyExist := search.NewQueryBuilder().AddMapQuery(fieldLabel, keyWrapper(keyValuePolicy.GetKey()), valueQuery).ProtoQuery()
	queryIfKeyDoesNotExist := search.NewQueryBuilder().AddMapQuery(fieldLabel, search.NegateQueryString(keyWrapper(keyValuePolicy.GetKey())), "").ProtoQuery()

	q = &v1.Query{
		Query: &v1.Query_Disjunction{
			Disjunction: &v1.DisjunctionQuery{
				Queries: []*v1.Query{queryIfKeyExist, queryIfKeyDoesNotExist},
			},
		},
	}

	v = func(_ context.Context, result search.Result) searchbasedpolicies.Violations {
		return searchbasedpolicies.Violations{
			AlertViolations: []*storage.Alert_Violation{
				{Message: fmt.Sprintf("Required %s not found (%s)", fieldName, printKeyValuePolicy(keyValuePolicy))},
			},
		}
	}
	return
}
