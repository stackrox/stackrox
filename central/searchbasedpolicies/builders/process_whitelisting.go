package builders

import (
	"github.com/stackrox/rox/central/searchbasedpolicies"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// ProcessWhitelistingBuilder is a wrapper for process whitelisting
type ProcessWhitelistingBuilder struct{}

// Query implements the PolicyQueryBuilder interface.
func (p ProcessWhitelistingBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	if fields.GetSetWhitelist() == nil || !fields.GetWhitelistEnabled() {
		return
	}

	q = search.NewQueryBuilder().AddStrings(search.DeploymentID, search.WildcardString).ProtoQuery()
	v = func(search.Result) searchbasedpolicies.Violations {
		return searchbasedpolicies.Violations{
			AlertViolations: []*storage.Alert_Violation{
				{
					Message: "Process whitelist has been violated",
				},
			},
		}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (ProcessWhitelistingBuilder) Name() string {
	return "Query builder for process whitelisting"
}
