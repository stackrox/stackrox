package mapping

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap contains fields which the indexer should index in a document
var OptionsMap = search.Walk(v1.SearchCategory_SUBJECTS, "subject", (*storage.Subject)(nil))
