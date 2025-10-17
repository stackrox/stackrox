package scancomponent

import (
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

// ComponentID creates a component ID from the given name and version and os.
func ComponentID(name, version, os string) string {
	return pgSearch.IDFromPks([]string{name, version, os})
}

// ComponentIDV2 creates a component ID from the given name and version and architecture and imageID.
func ComponentIDV2(component *storage.EmbeddedImageScanComponent, imageID string, index int) string {
	// The index it occurs in the component list is sufficient for uniqueness.  We do not need to be able to
	// rebuild this ID at query time from an embedded object.  Which is why we were forced to use a hash before.
	return pgSearch.IDFromPks([]string{component.GetName(), strconv.Itoa(index), imageID})
}
