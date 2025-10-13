package tlscheckcache

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/tlscheckcache"
)

var (
	once  sync.Once
	cache tlscheckcache.Cache
)

func initialize() {
	cache = tlscheckcache.New(
		tlscheckcache.WithMetricSubsystem(metrics.CentralSubsystem),
		tlscheckcache.WithTTL(env.RegistryTLSCheckTTL.DurationSetting()),
	)
}

// Singleton will return a single instance of the tls check cache to
// any callers.
func Singleton() tlscheckcache.Cache {
	once.Do(initialize)
	return cache
}
