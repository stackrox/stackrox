package sensor

import (
	"reflect"
	"sync"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"github.com/karlseguin/ccache"
)

func newPendingDeployments() *pendingDeploymentEvents {
	return &pendingDeploymentEvents{
		pending: ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(100)),
	}
}

// pendingDeploymentEvents is a simple thread safe (key, value) structure to hold all of the
// DeploymentsEventWraps while central is processing their deployments.
type pendingDeploymentEvents struct {
	mutex sync.Mutex

	pending *ccache.Cache
}

func (p *pendingDeploymentEvents) add(ew *listeners.DeploymentEventWrap) (isAlreadyPending bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	isAlreadyPending = p.checkAlreadyPresent(ew)
	p.pending.Set(ew.Deployment.GetId(), ew, time.Hour*1)
	return
}

func (p *pendingDeploymentEvents) remove(ew *listeners.DeploymentEventWrap) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.pending.Delete(ew.Deployment.GetId())
	return
}

func (p *pendingDeploymentEvents) fetch(deploymentID string) (ew *listeners.DeploymentEventWrap, exists bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	presentItem := p.pending.Get(deploymentID)
	if presentItem == nil {
		return nil, false
	}
	return presentItem.Value().(*listeners.DeploymentEventWrap), true
}

func (p *pendingDeploymentEvents) checkAlreadyPresent(ew *listeners.DeploymentEventWrap) bool {
	presentItem := p.pending.Get(ew.Deployment.GetId())
	if presentItem == nil {
		return false
	}

	deploymentEvent := presentItem.Value().(*listeners.DeploymentEventWrap)
	deployment := deploymentEvent.GetDeployment()

	deployment.UpdatedAt = ew.Deployment.GetUpdatedAt()
	deployment.Version = ew.Deployment.GetVersion()
	if !reflect.DeepEqual(deployment, ew.Deployment) {
		return false
	}

	return true
}
