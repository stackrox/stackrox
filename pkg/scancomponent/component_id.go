package scancomponent

import (
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log = logging.LoggerForModule()
)

// ComponentID creates a component ID from the given name and version (and os if postgres is enabled).
func ComponentID(name, version, os string) string {
	if features.PostgresDatastore.Enabled() {
		return postgres.IDFromPks([]string{name, version, os})
	}
	return edges.EdgeID{ParentID: name, ChildID: version}.ToString()
}

// IDToParts returns the parts—name, version, and operating system—that make up CVE ID.
func IDToParts(id string) (string, string, string) {
	if features.PostgresDatastore.Enabled() {
		parts := postgres.IDToParts(id)
		if len(parts) > 3 {
			log.Errorf("More than 3 parts found in component ID: %v", parts)
			return "", "", ""
		}

		switch len(parts) {
		case 0:
			return "", "", ""
		case 1:
			return parts[0], "", ""
		case 2:
			return parts[0], parts[1], ""
		default:
			return parts[0], parts[1], parts[2]
		}
	}

	parts, err := edges.FromString(id)
	if err != nil {
		log.Errorf("Failed to obtain component ID parts: %v", err)
		return "", "", ""
	}
	return parts.ParentID, parts.ChildID, ""
}
