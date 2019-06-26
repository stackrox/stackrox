package mapping

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// OptionsMap contains fields which the indexer should index in a document
var OptionsMap = blevesearch.Walk(v1.SearchCategory_SUBJECTS, "subject", (*storage.Subject)(nil))
