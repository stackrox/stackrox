package declarativeconfig

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Manager
)

// ReconciliationErrorReporter processes declarative resources reconciliation errors.
//
//go:generate mockgen-wrapper
type ReconciliationErrorReporter interface {
	ProcessError(protoValue proto.Message, err error)
}

type noOpErrorReporter struct{}

func (n noOpErrorReporter) ProcessError(m proto.Message, err error) {
	log.Warnf("Error: %v for message %v", err, m)
}

// ManagerSingleton provides the instance of Manager to use.
func ManagerSingleton(registry authproviders.Registry) Manager {
	once.Do(func() {
		instance = New(
			env.DeclarativeConfigReconcileInterval.DurationSetting(),
			env.DeclarativeConfigWatchInterval.DurationSetting(),
			updater.DefaultResourceUpdaters(registry),
			// TODO(ROX-15088): replace with actual health reporter
			noOpErrorReporter{})
	})
	return instance
}
