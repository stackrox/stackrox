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

	// LastUpgradeIDAnnotationKey is an annotation key for storing the ID of the last upgrade process that has modified
	// an object. This is used to inform the upgrader that an object no longer needs to be considered, even if it
	// appears different from the desired post-upgrade state.
	// The upgrader sets this on created or updated objects just before making a live change to the state of the
	// Kubernetes cluster, it is not part of any "have/want" state computation for the above reasons.
	LastUpgradeIDAnnotationKey = `sensor-upgrader.stackrox.io/last-upgrade-id`

	// PreserveResourcesAnnotationKey is an annotation key that when mapped to a "true" value instructs
	// the upgrader to preserve existing resource specifications.
	PreserveResourcesAnnotationKey = `auto-upgrade.stackrox.io/preserve-resources`
)
