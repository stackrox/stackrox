package mappings

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is exposed for e2e test
var OptionsMap = search.OptionsMapFromMap(map[search.FieldLabel]*v1.SearchField{
	search.PolicyID:       search.NewField(v1.SearchCategory_POLICIES, "policy.id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	search.Enforcement:    search.NewEnforcementField(v1.SearchCategory_POLICIES, "policy.enforcement_actions"),
	search.PolicyName:     search.NewStringField(v1.SearchCategory_POLICIES, "policy.name"),
	search.LifecycleStage: search.NewLifecycleField(v1.SearchCategory_POLICIES, "policy.lifecycle_stages"),
	search.Description:    search.NewStringField(v1.SearchCategory_POLICIES, "policy.description"),
	search.Category:       search.NewStringField(v1.SearchCategory_POLICIES, "policy.categories"),
	search.Severity:       search.NewSeverityField(v1.SearchCategory_POLICIES, "policy.severity"),
})
