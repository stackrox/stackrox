package postgres

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
)

// The groups table is generated without a search category, which leaves its schema without
// an options map. Queries over the group property fields cannot be compiled without one,
// even though the properties carry search tags, so register the fields here. This allows
// narrow lookups such as an exact match on auth provider, key, and value.
func init() {
	pkgSchema.GroupsSchema.SetOptionsMap(search.Walk(v1.SearchCategory_SEARCH_UNSET,
		"groupproperties", (*storage.GroupProperties)(nil)))
}
