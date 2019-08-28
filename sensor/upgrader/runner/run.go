package runner

import "github.com/stackrox/rox/sensor/upgrader/upgradectx"

// Run runs the upgrader.
func Run(ctx *upgradectx.UpgradeContext) error {
	r := &runner{ctx: ctx}
	return r.Run()
}
