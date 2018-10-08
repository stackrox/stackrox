package cache

import (
	"sync"
	"time"

	"github.com/deckarep/golang-set"
	"github.com/karlseguin/ccache"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var logger = logging.LoggerForModule()

func newContainerCache() *ContainerCache {
	return &ContainerCache{
		containerToDeployment:  ccache.New(ccache.Configure().MaxSize(10000).ItemsToPrune(500)),
		deploymentToContainers: make(map[string]mapset.Set),
	}
}

// ContainerCache is a simple thread safe (key, value) structure to hold all of the
// DeploymentsEventWraps while central is processing their deployments.
type ContainerCache struct {
	mutex sync.Mutex

	// "Reverse" cache for quick lookup of deployment ID by container ID
	containerToDeployment  *ccache.Cache
	deploymentToContainers map[string]mapset.Set
}

// This function creates a shortened Docker ID, which we should get rid once collector supports full shas
func toShortID(str string) string {
	if len(str) <= 12 {
		return str
	}
	return str[:12]
}

// AddDeployment adds a deployment to the cache
func (c *ContainerCache) AddDeployment(deployment *v1.Deployment) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.deploymentToContainers[deployment.GetId()] == nil {
		c.deploymentToContainers[deployment.GetId()] = mapset.NewSet()
	}

	for _, container := range deployment.GetContainers() {
		for _, instance := range container.GetInstances() {
			if instance.GetInstanceId().GetId() != "" {
				shortContainerID := toShortID(instance.GetInstanceId().GetId())

				// Add container to deployment mapping
				c.containerToDeployment.Set(shortContainerID, deployment.GetId(), time.Hour*1)

				// Add deployment to container mapping so that we can clean up efficiently
				c.deploymentToContainers[deployment.GetId()].Add(shortContainerID)
			}
		}
	}
}

// RemoveDeployment from the cache asynchronously
func (c *ContainerCache) RemoveDeployment(id string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	containerSet, ok := c.deploymentToContainers[id]
	if !ok {
		return
	}
	delete(c.deploymentToContainers, id)
	for _, s := range set.StringSliceFromSet(containerSet) {
		c.containerToDeployment.Delete(s)
	}
}

// GetDeploymentFromContainerID gets the deployment id for the passed container
func (c *ContainerCache) GetDeploymentFromContainerID(containerID string) (string, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	presentItem := c.containerToDeployment.Get(containerID)
	if presentItem == nil {
		return "", false
	}
	return presentItem.Value().(string), true
}
