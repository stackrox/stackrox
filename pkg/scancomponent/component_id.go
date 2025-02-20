package scancomponent

import (
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

// ComponentID creates a component ID from the given name and version and os.
func ComponentID(name, version, os string) string {
	return pgSearch.IDFromPks([]string{name, version, os})
}

// ComponentIDV2 creates a component ID from the given name and version and architecture and imageID.
func ComponentIDV2(name, version, architecture, imageID string) string {
	return pgSearch.IDFromPks([]string{name, version, architecture, imageID})
}
