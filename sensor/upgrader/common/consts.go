package common

import "github.com/stackrox/rox/pkg/namespaces"

const (
	// Namespace is the namespace in which the upgrader operates
	Namespace = namespaces.StackRox

	// UpgradeProcessIDLabelKey is the key of a label storing the ID of the current upgrade process. This is used to
	// ensure state does not get mixed up even if an upgrader process is terminated abruptly, e.g., by deleting the
	// deployment (but not other objects).
	UpgradeProcessIDLabelKey = `sensor-upgrader.stackrox.io/process-id`

	// UpgradeResourceLabelKey is a label key attached to all auto-upgradeable resources.
	UpgradeResourceLabelKey = `auto-upgrade.stackrox.io/component`
	// UpgradeResourceLabelValue is the label value for the above key that identifies resources from the sensor bundle.
	UpgradeResourceLabelValue = `sensor`
)
