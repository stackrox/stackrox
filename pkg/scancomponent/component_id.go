package scancomponent

import (
	"github.com/stackrox/stackrox/pkg/dackbox/edges"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/search/postgres"
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
