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
