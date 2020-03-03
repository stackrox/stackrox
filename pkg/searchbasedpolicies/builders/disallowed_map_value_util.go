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

func disallowedMapValueQuery(optionsMap map[search.FieldLabel]*v1.SearchField, keyValuePolicy *storage.KeyValuePolicy, fieldLabel search.FieldLabel, fieldName string, name string, keyWrapper func(string) string) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	_, err = getSearchFieldNotStored(fieldLabel, optionsMap)
	if err != nil {
		err = errors.Wrapf(err, "%s", name)
		return
	}

	var valueQuery string
	if keyValuePolicy.GetValue() == "" {
		valueQuery = search.WildcardString
	} else {
		valueQuery = search.RegexQueryString(keyValuePolicy.GetValue())
	}
	q = search.NewQueryBuilder().AddMapQuery(fieldLabel, keyWrapper(keyValuePolicy.GetKey()), valueQuery).ProtoQuery()

	v = func(_ context.Context, result search.Result) searchbasedpolicies.Violations {
		return searchbasedpolicies.Violations{
			AlertViolations: []*storage.Alert_Violation{
				{Message: fmt.Sprintf("Disallowed %s found (%s)", fieldName, printKeyValuePolicy(keyValuePolicy))},
			},
		}
	}
	return
}
