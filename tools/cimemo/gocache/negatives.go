package gocache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Negatives remembers actions known to be absent from the cloud so we
// stop paying a round trip (and a timeout risk) for them every
// session. Ported from gocache-serve's phantom entries. Persisted in
// the cache dir (go ignores foreign files there; `go clean -cache`
// removing it is harmless — worst case we re-probe).
type Negatives struct {
	mu   sync.Mutex
	set  map[string]bool
	path string
}

const negativesMax = 10000

func loadNegatives(cacheDir string) *Negatives {
	n := &Negatives{
		set:  make(map[string]bool),
		path: filepath.Join(cacheDir, "cimemo-negatives.json"),
	}
	data, err := os.ReadFile(n.path)
	if err == nil {
		var ids []string
		if json.Unmarshal(data, &ids) == nil {
			for _, id := range ids {
				n.set[id] = true
			}
		}
	}
	return n
}

func (n *Negatives) Has(id string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.set[id]
}

// Add records a known-absent action (bounded: a huge negative set
// means the store changed and re-probing is cheaper than carrying it).
func (n *Negatives) Add(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.set) < negativesMax {
		n.set[id] = true
	}
}

func (n *Negatives) Clear(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.set, id)
}

func (n *Negatives) Persist() {
	n.mu.Lock()
	defer n.mu.Unlock()
	ids := make([]string, 0, len(n.set))
	for id := range n.set {
		ids = append(ids, id)
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return
	}
	os.WriteFile(n.path, data, 0o644)
}
