package snapshot

import (
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

// Options controls the operation of the snapshotter.
type Options struct {
	Store     bool
	MustExist bool
}

// TakeOrReadSnapshot either reads a previously snapshotted pre-upgrade state from the secret, or creates a secret with this
// state.
func TakeOrReadSnapshot(ctx *upgradectx.UpgradeContext, opts Options) ([]k8sutil.Object, error) {
	s := &snapshotter{ctx: ctx, opts: opts}
	return s.SnapshotState()
}
