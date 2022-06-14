package bundle

import (
	"github.com/stackrox/stackrox/pkg/k8sutil"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/sensor/upgrader/upgradectx"
)

var (
	log = logging.LoggerForModule()
)

// InstantiateBundle takes a view of the bundle context and instantiates all contained Kubernetes objects.
func InstantiateBundle(ctx *upgradectx.UpgradeContext, bundleContents Contents) ([]k8sutil.Object, error) {
	inst := &instantiator{ctx: ctx}
	return inst.Instantiate(bundleContents)
}
