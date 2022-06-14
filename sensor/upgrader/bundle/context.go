package bundle

import "github.com/stackrox/stackrox/pkg/k8sutil"

// upgradeContext is a trimmed version of *upgradectx.UpgradeContext to facilitate unit testing.
// TODO: usages of *upgradectx.UpgradeContext should be converted to an interface everywhere.
type upgradeContext interface {
	ParseAndValidateObject(data []byte) (k8sutil.Object, error)
	InCertRotationMode() bool
}
