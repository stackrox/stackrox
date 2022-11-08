package generate

import (
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/renderer"
)

var (
	cfg renderer.Config
)

func validateConfig(c *renderer.Config) error {
	if c.HostPath == nil {
		return nil
	}
	return validateHostPathInstance(c.HostPath.DB)
}

func validateHostPathInstance(instance *renderer.HostPathPersistenceInstance) error {
	if instance == nil {
		return nil
	}
	if instance.HostPath == "" {
		return errox.InvalidArgs.New("non-empty HostPath must be specified")
	}
	if (instance.NodeSelectorKey == "") != (instance.NodeSelectorValue == "") {
		return errox.InvalidArgs.New("both node selector key and node selector value must be specified when using a hostpath")
	}
	return nil
}
