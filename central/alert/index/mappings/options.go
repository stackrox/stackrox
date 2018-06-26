package mappings

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
)

// OptionsMap is exposed for e2e test.
var OptionsMap = map[string]*v1.SearchField{
	search.Violation: search.NewStringField("alert.violations.message"),
	search.Stale:     search.NewBoolField("alert.stale"),
}
