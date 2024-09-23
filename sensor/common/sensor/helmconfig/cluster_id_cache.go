package helmconfig

import (
	"os"
)

const (
	clusterIDCacheFile = `/var/cache/stackrox/cluster-id`
)

// StoreCachedClusterID stores the cluster ID in the filesystem cache.
func StoreCachedClusterID(id string) error {
	return os.WriteFile(clusterIDCacheFile, []byte(id+"\n"), 0644)
}
