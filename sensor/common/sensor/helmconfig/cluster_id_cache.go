package helmconfig

import (
	"bytes"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	clusterIDCacheFile = `/var/cache/stackrox/cluster-id`
)

// StoreCachedClusterID stores the cluster ID in the filesystem cache.
func StoreCachedClusterID(id string) error {
	if err := os.WriteFile(clusterIDCacheFile, []byte(id+"\n"), 0644); err != nil {
		return errors.Wrapf(err, "writing cluster ID to cache file %s", clusterIDCacheFile)
	}
	return nil
}

// LoadCachedClusterID loads a cached cluster ID from the filesystem cache.
func LoadCachedClusterID() (string, error) {
	cachedIDBytes, err := os.ReadFile(clusterIDCacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", errors.Wrapf(err, "read cluster ID cache file %s", clusterIDCacheFile)
	}
	id := string(bytes.TrimSpace(cachedIDBytes))
	if id == "" {
		return "", nil
	}
	if _, err := uuid.FromString(id); err != nil {
		return "", errors.Wrapf(err, "file %s contains invalid contents", clusterIDCacheFile)
	}
	return id, nil
}
