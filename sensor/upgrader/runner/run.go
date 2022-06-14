package runner

import "github.com/stackrox/stackrox/sensor/upgrader/upgradectx"

// Run runs the given workflow in the upgrader.
func Run(ctx *upgradectx.UpgradeContext, workflow string) error {
	r, err := newRunner(ctx, workflow)
	if err != nil {
		return err
	}
	return r.runFullWorkflow()
}
