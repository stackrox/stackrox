package env

import "time"

var (
	// PermissionTimeout will set the duration for which we will cache a user's permissions
	PermissionTimeout = registerDurationSetting("ROX_SAC_PERMISSION_CACHE_TTL", time.Minute*10)
)
