package internal

import "github.com/stackrox/rox/pkg/env"

var (
	dbMountPathSetting = env.RegisterSetting("ROX_DB_MOUNT_PATH",
		env.WithDefault("/var/lib/stackrox"))
)

// DBMountPath returns the directory path (within a container) where database storage device is mounted.
func DBMountPath() string {
	return dbMountPathSetting.Setting()
}
