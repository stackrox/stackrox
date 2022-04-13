package plugins

import (
	"github.com/stackrox/stackrox/central/externalbackups/plugins/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
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
