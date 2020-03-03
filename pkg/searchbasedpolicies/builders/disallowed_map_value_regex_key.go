package builders

import (
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

// DisallowedMapValueRegexKeyQueryBuilder builds queries to check for the existence of a map value.
type DisallowedMapValueRegexKeyQueryBuilder struct {
	GetKeyValuePolicy func(*storage.PolicyFields) *storage.KeyValuePolicy
	FieldName         string
	FieldLabel        search.FieldLabel
}

// Query implements the PolicyQueryBuilder interface.
func (r DisallowedMapValueRegexKeyQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	keyValuePolicy := r.GetKeyValuePolicy(fields)
	if keyValuePolicy.GetKey() == "" {
		if keyValuePolicy.GetValue() != "" {
			err = fmt.Errorf("key value policy for %s had no key, only a value: %s", r.FieldName, keyValuePolicy.GetValue())
			return
		}
		return
	}

	return disallowedMapValueQuery(optionsMap, keyValuePolicy, r.FieldLabel, r.FieldName, r.Name(), search.RegexQueryString)
}

// Name implements the PolicyQueryBuilder interface.
func (r DisallowedMapValueRegexKeyQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for disallowed map value %s", r.FieldName)
}
