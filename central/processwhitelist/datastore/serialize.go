package datastore

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

type keyPrefix string

const (
	deploymentContainerKeyPrefix keyPrefix = "DC"
)

func keyToID(key *storage.ProcessWhitelistKey) (string, error) {
	if stringutils.AllNotEmpty(key.GetClusterId(), key.GetNamespace(), key.GetDeploymentId(), key.GetContainerName()) {
		return fmt.Sprintf("%s:%s:%s:%s:%s", deploymentContainerKeyPrefix, key.GetClusterId(), key.GetNamespace(), key.GetDeploymentId(), key.GetContainerName()), nil
	}
	return "", fmt.Errorf("invalid key %+v: doesn't match any of our known patterns", key)
}
