package deploy

import (
	"fmt"

	"github.com/stackrox/rox/roxctl/central/deploy/renderer"
)

var (
	cfg renderer.Config
)

func validateConfig(c renderer.Config) error {
	return validateHostPath(c.HostPath)
}

func validateHostPath(hostpath *renderer.HostPathPersistence) error {
	if hostpath == nil {
		return nil
	}
	if (hostpath.NodeSelectorKey == "") != (hostpath.NodeSelectorValue == "") {
		return fmt.Errorf("Both node selector key and node selector value must be specified when using a hostpath")
	}
	return nil
}
