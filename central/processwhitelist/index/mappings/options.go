package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// OptionsMap is exposed for e2e test
var OptionsMap = blevesearch.Walk(v1.SearchCategory_PROCESS_WHITELISTS, "process_whitelist", (*storage.ProcessWhitelist)(nil))
