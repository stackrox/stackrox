package mappings

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// OptionsMap is the map of indexed fields in secret and relationship objects.
var OptionsMap = search.Walk(v1.SearchCategory_SECRETS, "secret", (*storage.Secret)(nil))
