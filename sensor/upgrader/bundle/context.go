package bundle

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// upgradeContext is a trimmed version of *upgradectx.UpgradeContext to facilitate unit testing.
// TODO: usages of *upgradectx.UpgradeContext should be converted to an interface everywhere.
type upgradeContext interface {
	ParseAndValidateObject(data []byte) (*unstructured.Unstructured, error)
	InCertRotationMode() bool
	IsPodSecurityEnabled() bool
}
