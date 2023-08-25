package certdistribution

import "github.com/stackrox/rox/pkg/env"

var (
	// cacheDir is the directory in which certificates to be distributed are stored.
	cacheDir = env.RegisterSetting("ROX_CERTIFICATE_CACHE_DIR", env.WithDefault("/var/cache/stackrox/.certificates"))
)
