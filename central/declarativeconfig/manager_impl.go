package declarativeconfig

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	watchInterval        = 5 * time.Second
	declarativeConfigDir = "/run/stackrox.io/declarative-configuration"
)

type managerImpl struct {
	once sync.Once
}

// New creates a new instance of Manager.
// Note that it will not watch the declarative configuration directories when created, only after
// WatchDeclarativeConfigDir has been called.
func New() Manager {
	return &managerImpl{}
}

func (m *managerImpl) WatchDeclarativeConfigDir() {
	m.once.Do(func() {
		wh := &watchHandler{m: m}
		// Set Force to true, so we explicitly retry watching the files within the directory and not stop on the first
		// error occurred.
		watchOpts := k8scfgwatch.Options{Interval: watchInterval, Force: true}
		_ = k8scfgwatch.WatchConfigMountDir(context.Background(), declarativeConfigDir,
			k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)
	})
}
