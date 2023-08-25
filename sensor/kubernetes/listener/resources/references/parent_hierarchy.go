package references

import (
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ParentHierarchy defines the interface for managing dependencies between deployments
type ParentHierarchy interface {
	Add(obj metav1.Object)
	AddManually(objUID string, parentUID string)
	Remove(id string)
	IsValidChild(parent string, child metav1.Object) bool
	TopLevelParents(child string) set.StringSet
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

func isValidOwnerRef(ref metav1.OwnerReference) bool {
	return ref.UID != "" && resources.IsTrackedOwnerReference(ref)
}

func (p *parentHierarchy) Add(obj metav1.Object) {
	parents := make([]string, 0, len(obj.GetOwnerReferences()))
	for _, ref := range obj.GetOwnerReferences() {
		if isValidOwnerRef(ref) {
			// Only bother adding parents we track.
			parents = append(parents, string(ref.UID))
		}
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	p.parents[string(obj.GetUID())] = parents
}

func (p *parentHierarchy) AddManually(objUID string, parentUID string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.parents[objUID] = append(p.parents[objUID], parentUID)
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

func (p *parentHierarchy) IsValidChild(parent string, child metav1.Object) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, ref := range child.GetOwnerReferences() {
		if isValidOwnerRef(ref) {
			// Only bother checking for parents we track.
			if p.searchRecursiveNoLock(parent, string(ref.UID)) {
				return true
			}
		}
	}
	return false
}

func (p *parentHierarchy) addTopLevelParentsRecursiveNoLock(child string, parents set.StringSet) {
	currParents := p.parents[child]
	if len(currParents) == 0 {
		parents.Add(child)
	} else {
		for _, currParent := range currParents {
			p.addTopLevelParentsRecursiveNoLock(currParent, parents)
		}
	}
}

func (p *parentHierarchy) TopLevelParents(child string) set.StringSet {
	p.lock.RLock()
	defer p.lock.RUnlock()

	parents := set.NewStringSet()
	p.addTopLevelParentsRecursiveNoLock(child, parents)
	return parents
}
