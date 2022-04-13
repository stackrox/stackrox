package datastore

import (
	"fmt"
	"strings"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

type keyPrefix string

const (
	deploymentContainerKeyPrefix keyPrefix = "DC"
)

func keyToID(key *storage.ProcessBaselineKey) (string, error) {
	if stringutils.AllNotEmpty(key.GetClusterId(), key.GetNamespace(), key.GetDeploymentId(), key.GetContainerName()) {
		return fmt.Sprintf("%s:%s:%s:%s:%s", deploymentContainerKeyPrefix, key.GetClusterId(), key.GetNamespace(), key.GetDeploymentId(), key.GetContainerName()), nil
	}
	return "", fmt.Errorf("invalid key %+v: doesn't match any of our known patterns", key)
}

// IDToKey converts a string process baseline key to its proto object.
func IDToKey(id string) (*storage.ProcessBaselineKey, error) {
	if strings.HasPrefix(id, string(deploymentContainerKeyPrefix)) {
		keys := strings.Split(id, ":")
		if len(keys) == 5 {
			resKey := &storage.ProcessBaselineKey{
				ClusterId:     keys[1],
				Namespace:     keys[2],
				DeploymentId:  keys[3],
				ContainerName: keys[4],
			}

			return resKey, nil
		}
	}

	return nil, fmt.Errorf("invalid id %s: doesn't match any of our known patterns", id)
}
