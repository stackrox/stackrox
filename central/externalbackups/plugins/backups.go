package plugins

import (
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

type creator func(backup *storage.ExternalBackup) (types.ExternalBackup, error)

var (
	// Registry holds a map from name of the external backup to a creator of that type
	Registry = make(map[string]creator)

	log = logging.LoggerForModule()
)

// Add adds a new external backup to the registry
func Add(name string, creator creator) {
	if _, ok := Registry[name]; ok {
		log.Fatalf("external backup %q is already registered", name)
	}
	Registry[name] = creator
}
