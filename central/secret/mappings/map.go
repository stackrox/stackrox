package mappings

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// OptionsMap is the map of indexed fields in secret and relationship objects.
var OptionsMap = search.Walk(v1.SearchCategory_SECRETS, "secret", (*storage.Secret)(nil))
