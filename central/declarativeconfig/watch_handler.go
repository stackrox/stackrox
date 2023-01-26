package declarativeconfig

import "github.com/stackrox/rox/pkg/logging"

// TODO(ROX-14147): Add reconciliation of declarative configuration to the watch handler.
var (
	log = logging.LoggerForModule()
)

type watchHandler struct {
	m *managerImpl
}

func (w *watchHandler) OnChange(dir string) (interface{}, error) {
	return nil, nil
}

func (w *watchHandler) OnStableUpdate(val interface{}, err error) {
}

func (w *watchHandler) OnWatchError(err error) {
	log.Errorf("Error watching declarative configuration directory: %+v", err)
}
