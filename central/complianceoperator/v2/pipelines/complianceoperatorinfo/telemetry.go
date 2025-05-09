package complianceoperatorinfo

import (
	"context"
	"strconv"
	"sync/atomic"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	cot  *CoTelemetry
	once sync.Once
)

// TelemetrySingleton returns the telemetry object for complianceoperatorinfo
func TelemetrySingleton() *CoTelemetry {
	once.Do(func() {
		cot = &CoTelemetry{}
	})
	return cot
}

type telemetry interface {
	TrackVersion(version string) bool
	GetVersion() (string, bool)
}

type CoTelemetry struct {
	currentValue string
	installed    atomic.Bool
}

func (c *CoTelemetry) TrackVersion(version string) bool {
	changed := false
	if !c.installed.Load() {
		c.installed.Store(true)
		changed = true
	}
	if c.currentValue != version {
		c.currentValue = version
		return true
	}
	return changed
}

func (c *CoTelemetry) GetVersion() (string, bool) {
	if !c.installed.Load() {
		return "unknown", false
	}
	// Operator is installed, but the version string is empty
	if c.currentValue == "" {
		return "unknown", true
	}
	return c.currentValue, true
}

func Gather() phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.ComplianceOperator)))
		version, installed := TelemetrySingleton().GetVersion()
		return map[string]any{
			"Compliance Operator Version":   version,
			"Compliance Operator Installed": strconv.FormatBool(installed),
		}, nil
	}
}
