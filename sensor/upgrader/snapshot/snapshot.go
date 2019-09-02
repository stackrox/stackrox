package snapshot

import (
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

// TakeOrReadSnapshot either reads a previously snapshotted pre-upgrade state from the secret, or creates a secret with this
// state.
func TakeOrReadSnapshot(ctx *upgradectx.UpgradeContext, store bool) ([]k8sobjects.Object, error) {
	s := &snapshotter{ctx: ctx, store: store}
	return s.SnapshotState()
}
