package sensor

import (
	"sync"
	"time"

	"github.com/karlseguin/ccache"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
)

var logger = logging.LoggerForModule()

func newPendingEvents() *pendingEvents {
	return &pendingEvents{
		pending:               ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(100)),
		containerToDeployment: ccache.New(ccache.Configure().MaxSize(5000).ItemsToPrune(500)),
	}
}

// pendingEvents is a simple thread safe (key, value) structure to hold all of the
// DeploymentsEventWraps while central is processing their deployments.
type pendingEvents struct {
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

func (p *pendingEvents) add(ew *listeners.EventWrap) bool {
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

func (p *pendingEvents) remove(ew *listeners.EventWrap) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, container := range ew.GetDeployment().GetContainers() {
		p.containerToDeployment.Delete(container.GetId())
	}
	p.pending.Delete(ew.GetId())
	return
}

func (p *pendingEvents) fetch(deploymentID string) (ew *listeners.EventWrap, exists bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	presentItem := p.pending.Get(deploymentID)
	if presentItem == nil {
		return nil, false
	}
	return presentItem.Value().(*listeners.EventWrap), true
}

func (p *pendingEvents) fetchDeploymentIDFromContainerID(containerID string) (deploymentID string, exists bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	presentItem := p.containerToDeployment.Get(containerID)
	if presentItem == nil {
		return "", false
	}
	return presentItem.Value().(string), true
}

func (p *pendingEvents) checkAlreadyPresent(event *listeners.EventWrap) bool {
	presentItem := p.pending.Get(event.GetId())
	if presentItem == nil {
		return false
	}
	cachedEvent := presentItem.Value().(*listeners.EventWrap)
	return cachedEvent.Equals(event)
}
