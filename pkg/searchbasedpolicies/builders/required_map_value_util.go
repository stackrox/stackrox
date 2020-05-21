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

const (
	// RequiredKeyValuePrefix is the prefix to a map query to denote that the specific kv pair is required,
	// and any map without the key value should be flagged.
	// This is incredibly hacky, and is only done this way because search-based policies are going away soon.
	RequiredKeyValuePrefix = "REQUIRE\t"
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

	value := keyValuePolicy.GetValue()
	if value != "" {
		value = search.RegexQueryString(value)
	}

	q = search.NewQueryBuilder().AddMapQuery(fieldLabel, RequiredKeyValuePrefix+keyWrapper(keyValuePolicy.GetKey()), value).ProtoQuery()

	v = func(_ context.Context, result search.Result) searchbasedpolicies.Violations {
		return searchbasedpolicies.Violations{
			AlertViolations: []*storage.Alert_Violation{
				{Message: fmt.Sprintf("Required %s not found (%s)", fieldName, printKeyValuePolicy(keyValuePolicy))},
			},
		}
	}
	return
}
