package types

import (
	"github.com/stackrox/rox/pkg/reconcile"
)

type PodStore interface {
	reconcile.Reconcilable
	Add(string, string, string)
	Remove(string, string, string)
	Get(string) Pod
	Clean()
}

type Pod interface {
	GetNamespace() string
	GetUUID() string
	GetDeploymentID() string
}

type pod struct {
	namespace    string
	uuid         string
	deploymentId string
}

func (p *pod) GetNamespace() string {
	if p == nil {
		return ""
	}
	return p.namespace
}

func (p *pod) GetUUID() string {
	if p == nil {
		return ""
	}
	return p.uuid
}

func (p *pod) GetDeploymentID() string {
	if p == nil {
		return ""
	}
	return p.deploymentId
}

var _ Pod = (*pod)(nil)

type podStore struct {
	// namesapce -> uuid -> deploymentid
	pods map[string]map[string]*pod
}

func NewPodStore() PodStore {
	return &podStore{
		pods: make(map[string]map[string]*pod),
	}
}

func (p *podStore) Get(uuid string) Pod {
	for _, ns := range p.pods {
		for podUuid, p := range ns {
			if podUuid == uuid {
				return p
			}
		}
	}
	return nil
}

func (p *podStore) ReconcileDelete(resType, resID string, resHash uint64) ([]reconcile.Resource, error) {
	//TODO implement me
	return nil, nil
}

func (p *podStore) Add(ns string, uuid string, deploymentId string) {
	if _, ok := p.pods[ns]; !ok {
		p.pods[ns] = make(map[string]*pod)
	}
	p.pods[ns][uuid] = &pod{
		namespace:    ns,
		uuid:         uuid,
		deploymentId: deploymentId,
	}
}

func (p *podStore) Remove(ns string, uuid string, _ string) {
	if _, ok := p.pods[ns]; ok {
		delete(p.pods[ns], uuid)
	}
	if len(p.pods[ns]) == 0 {
		delete(p.pods, ns)
	}
}

func (p *podStore) Clean() {
	p.pods = make(map[string]map[string]*pod)
}

var _ PodStore = (*podStore)(nil)
