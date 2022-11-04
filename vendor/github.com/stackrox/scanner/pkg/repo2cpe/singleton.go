package repo2cpe

import (
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once     sync.Once
	instance *Mapping
)

// Singleton returns the cache instance to use.
func Singleton() *Mapping {
	once.Do(func() {
		instance = NewMapping()

		if definitionsDir := os.Getenv("REPO_TO_CPE_DIR"); definitionsDir != "" {
			log.Info("Loading repo-to-cpe map into mem")
			utils.Must(instance.Load(definitionsDir))
			log.Info("Done loading repo-to-cpe map into mem")
		}
	})
	return instance
}
