package mappings

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test
var OptionsMap = map[string]*v1.SearchField{
	search.PolicyID:    search.NewField("policy.id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	search.Enforcement: search.NewEnforcementField("policy.enforcement"),
	search.PolicyName:  search.NewStringField("policy.name"),
	search.Description: search.NewStringField("policy.description"),
	search.Category:    search.NewStringField("policy.categories"),
	search.Severity:    search.NewSeverityField("policy.severity"),
}
