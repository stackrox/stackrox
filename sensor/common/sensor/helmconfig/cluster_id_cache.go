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
	return os.WriteFile(clusterIDCacheFile, []byte(id+"\n"), 0644)
}

// LoadCachedClusterID loads a cached cluster ID from the filesystem cache.
func LoadCachedClusterID() (string, error) {
	cachedIDBytes, err := os.ReadFile(clusterIDCacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
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
