package declarativeconfig

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	watchInterval        = 5 * time.Second
	declarativeConfigDir = "/run/stackrox.io/declarative-configuration"
)

var (
	// Set Force to true, so we explicitly retry watching the files within the directory and not stop on the first
	// error occurred.
	watchOpts = k8scfgwatch.Options{Interval: watchInterval, Force: true}
)

type managerImpl struct {
	once sync.Once
	t    transform.Transformer
}

// New creates a new instance of Manager.
// Note that it will not watch the declarative configuration directories when created, only after
// WatchDeclarativeConfigDir has been called.
func New() Manager {
	return &managerImpl{
		t: transform.New(),
	}
}

func (m *managerImpl) WatchDeclarativeConfigDir() {
	m.once.Do(func() {
		// For each directory within the declarative configuration path, create a watch handler.
		// The reason we need multiple watch handlers and cannot simply watch the root directory is that
		// changes to directories are ignored within the watch handler.
		entries, err := os.ReadDir(declarativeConfigDir)
		if err != nil {
			if os.IsNotExist(err) {
				log.Info("Declarative configuration directory does not exist, no reconciliation will be done")
				return
			}
			utils.Should(err)
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			log.Infof("Start watch handler for declarative configuration for path %s",
				path.Join(declarativeConfigDir, entry.Name()))
			wh := newWatchHandler(m)
			_ = k8scfgwatch.WatchConfigMountDir(context.Background(), declarativeConfigDir,
				k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)
		}
	})
}

// ReconcileDeclarativeConfigs will take the file contents and transform these to declarative configurations.
// TODO(ROX-14693): Add upserting transformed resources.
// TODO(ROX-14694): Add deletion of resources.
func (m *managerImpl) ReconcileDeclarativeConfigs(contents [][]byte) {
	// No-op, see the TODOs within the function comment.
}
