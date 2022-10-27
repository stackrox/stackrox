package fake

import (
	"math/rand"

	"github.com/stackrox/rox/pkg/sync"
)

var (
	labelsPool = newLabelsPool()
)

type labelsPoolPerNamespace struct {
	pool        map[string][]map[string]string
	matchLabels bool
	lock        sync.RWMutex
}

func newLabelsPool() *labelsPoolPerNamespace {
	p := &labelsPoolPerNamespace{
		pool: make(map[string][]map[string]string),
	}
	return p
}

func (p *labelsPoolPerNamespace) add(namespace string, labels map[string]string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.matchLabels {
		p.pool[namespace] = append(p.pool[namespace], labels)
	}
}

func (p *labelsPoolPerNamespace) randomElem(namespace string) map[string]string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if !p.matchLabels {
		return createRandMap(16, 3)
	}
	labelsSlice, ok := p.pool[namespace]
	if !ok {
		return createRandMap(16, 3)
	}
	return labelsSlice[rand.Intn(len(labelsSlice))]
}
