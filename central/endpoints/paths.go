package endpoints

import (
	"path/filepath"

	"github.com/stackrox/rox/pkg/env"
)

const (
	endpointsConfigFile = `endpoints.yaml`
)

var (
	endpointsConfigDirSetting = env.RegisterSetting("ROX_ENDPOINTS_CONFIG_DIR",
		env.WithDefault("/etc/stackrox.d/endpoints"))
)

func endpointsConfigPath() string {
	return filepath.Join(endpointsConfigDirSetting.Setting(), endpointsConfigFile)
}
