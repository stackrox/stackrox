package snapshot

import (
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Options controls the operation of the snapshotter.
type Options struct {
	Store     bool
	MustExist bool
}

// TakeOrReadSnapshot either reads a previously snapshotted pre-upgrade state from the secret, or creates a secret with this
// state.
func TakeOrReadSnapshot(ctx *upgradectx.UpgradeContext, opts Options) ([]*unstructured.Unstructured, error) {
	s := &snapshotter{ctx: ctx, opts: opts}
	return s.SnapshotState()
}
