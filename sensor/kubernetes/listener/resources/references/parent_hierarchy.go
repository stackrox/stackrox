package references

import (
	"github.com/stackrox/rox/pkg/sync"
)

// ParentHierarchy defines the interface for managing dependencies between deployments
type ParentHierarchy interface {
	Add(parents []string, child string)
	Remove(id string)
	IsValidChild(parent string, child string) bool
}

type parentHierarchy struct {
	lock    sync.RWMutex
	parents map[string][]string
}

// NewParentHierarchy initializes a hierarchy to manage child parent relationships
func NewParentHierarchy() ParentHierarchy {
	return &parentHierarchy{
		parents: make(map[string][]string),
	}
}

func (p *parentHierarchy) Add(parents []string, child string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.parents[child] = parents
}

func (p *parentHierarchy) Remove(id string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.parents, id)
}

func (p *parentHierarchy) searchRecursiveNoLock(parent, child string) bool {
	if parent == child {
		return true
	}
	for _, currParent := range p.parents[child] {
		if p.searchRecursiveNoLock(parent, currParent) {
			return true
		}
	}
	return false
}

func (p *parentHierarchy) IsValidChild(parent, child string) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.searchRecursiveNoLock(parent, child)
}
