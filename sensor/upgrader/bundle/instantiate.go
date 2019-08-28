package bundle

import (
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

// InstantiateBundle takes a view of the bundle context and instantiates all contained Kubernetes objects.
func InstantiateBundle(ctx *upgradectx.UpgradeContext, bundleContents Contents) ([]k8sobjects.Object, error) {
	inst := &instantiator{ctx: ctx}
	return inst.Instantiate(bundleContents)
}
