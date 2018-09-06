package cache

import (
	"sync"
	"time"

	"github.com/karlseguin/ccache"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
)

var logger = logging.LoggerForModule()

func newPendingEvents() *PendingEvents {
	return &PendingEvents{
		pending:               ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(100)),
		containerToDeployment: ccache.New(ccache.Configure().MaxSize(5000).ItemsToPrune(500)),
	}
}

// PendingEvents is a simple thread safe (key, value) structure to hold all of the
// DeploymentsEventWraps while central is processing their deployments.
type PendingEvents struct {
	mutex sync.Mutex

	pending *ccache.Cache

	// "Reverse" cache for quick lookup of deployment ID by container ID
	containerToDeployment *ccache.Cache
}

// This function creates a shortened Docker ID, which we should get rid once collector supports full shas
func toShortID(str string) string {
	if len(str) <= 12 {
		return str
	}
	return str[:12]
}

// AddDeployment adds a deployment to the cache
func (p *PendingEvents) AddDeployment(ew *listeners.EventWrap) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if exactMatchIsPending := p.checkAlreadyPresent(ew); exactMatchIsPending {
		return false
	}

	for _, container := range ew.GetDeployment().GetContainers() {
		for _, instance := range container.GetInstances() {
			if instance.GetInstanceId().GetId() != "" {
				shortContainerID := toShortID(instance.GetInstanceId().GetId())
				p.containerToDeployment.Set(shortContainerID, ew.GetDeployment().GetId(), time.Hour*1)
			}
		}
	}
	p.pending.Set(ew.GetId(), ew, time.Hour*1)

	return true
}

// RemoveDeployment from the cache
func (p *PendingEvents) RemoveDeployment(ew *listeners.EventWrap) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, container := range ew.GetDeployment().GetContainers() {
		p.containerToDeployment.Delete(container.GetId())
	}
	p.pending.Delete(ew.GetId())
	return
}

// FetchDeployment from cache
func (p *PendingEvents) FetchDeployment(deploymentID string) (ew *listeners.EventWrap, exists bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	presentItem := p.pending.Get(deploymentID)
	if presentItem == nil {
		return nil, false
	}
	return presentItem.Value().(*listeners.EventWrap), true
}

// FetchDeploymentByContainer gets the deployment id for the passed container
func (p *PendingEvents) FetchDeploymentByContainer(containerID string) (deploymentID string, exists bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	presentItem := p.containerToDeployment.Get(containerID)
	if presentItem == nil {
		return "", false
	}
	return presentItem.Value().(string), true
}

func (p *PendingEvents) checkAlreadyPresent(event *listeners.EventWrap) bool {
	presentItem := p.pending.Get(event.GetId())
	if presentItem == nil {
		return false
	}
	cachedEvent := presentItem.Value().(*listeners.EventWrap)
	return cachedEvent.Equals(event)
}
