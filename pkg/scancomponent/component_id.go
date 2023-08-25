package scancomponent

import (
	"github.com/stackrox/rox/pkg/logging"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log = logging.LoggerForModule()
)

// ComponentID creates a component ID from the given name and version (and os if postgres is enabled).
func ComponentID(name, version, os string) string {
	return pgSearch.IDFromPks([]string{name, version, os})
}
