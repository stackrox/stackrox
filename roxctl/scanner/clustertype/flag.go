package clustertype

import (
	"github.com/spf13/pflag"
	"github.com/stackrox/stackrox/generated/storage"
)

var (
	w = wrapper{}
)

// Get returns the value that will be set by a flag that is passed the Value below.
// It WILL panic unless you can Value first.
func Get() storage.ClusterType {
	return *w.ClusterType
}

// Value returns the cluster type as a cobra value. Whatever that value is set to,
// it can be retrieved using Get.
// The caller must specify the default.
func Value(defaultClusterType storage.ClusterType) pflag.Value {
	w.ClusterType = &defaultClusterType
	return w
}
