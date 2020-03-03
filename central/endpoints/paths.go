package endpoints

import "path/filepath"

const (
	endpointsConfigDir  = `/etc/stackrox.d/endpoints`
	endpointsConfigFile = `endpoints.yaml`
)

var (
	endpointsConfigPath = filepath.Join(endpointsConfigDir, endpointsConfigFile)
)
