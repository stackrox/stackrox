package bundle

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	log = logging.LoggerForModule()
)

// InstantiateBundle takes a view of the bundle context and instantiates all contained Kubernetes objects.
func InstantiateBundle(ctx *upgradectx.UpgradeContext, bundleContents Contents) ([]*unstructured.Unstructured, error) {
	inst := &instantiator{ctx: ctx}
	return inst.Instantiate(bundleContents)
}
