package baseimage

import (
	"strings"
	"sync/atomic"

	"github.com/stackrox/rox/pkg/sync"
)

// Normalize layer digests (e.g., ensure "sha256:" prefix, lowercase, etc.).
func normalizeDigest(d string) string {
	return strings.ToLower(strings.TrimSpace(d))
}

// Node represents one layer in the chain.
type Node struct {
	// layer digest at this node; root has empty digest
	digest string

	// children keyed by next-layer digest
	children map[string]*Node

	// images that terminate exactly at this node (i.e., this path is a full image)
	images []ImageMeta

	// optional counters/metrics (atomic if heavily updated)
	// inserts int64

	// Per-node lock could be used for fine-grained concurrency,
	// but we keep a trie-level lock for simplicity & safety.
}

// Trie is the thread-safe prefix tree over layer digests.
type Trie struct {
	root *Node
	mu   sync.RWMutex
	size int64
}

func NewTrie() *Trie {
	return &Trie{
		root: &Node{
			digest:   "",
			children: make(map[string]*Node),
		},
		size: 0,
	}
}

// InsertImage inserts an image’s layer chain (ordered base->top).
// layers should be the layer digests as they appear in the image manifest's "layers" array.
func (t *Trie) InsertImage(layers []string, meta ImageMeta) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cur := t.root
	for _, raw := range layers {
		d := normalizeDigest(raw)
		next, ok := cur.children[d]
		if !ok {
			next = &Node{
				digest:   d,
				children: make(map[string]*Node),
			}
			cur.children[d] = next
		}
		cur = next
	}
	// Mark that an image ends here.
	cur.images = append(cur.images, meta)
	atomic.AddInt64(&t.size, 1)
}

// LongestPrefix finds the deepest node matching the given layer chain.
// It returns the depth (how many layers matched), and any images that exactly end at that node.
// If depth==0, only the root matched (i.e., no base).
func (t *Trie) LongestPrefix(layers []string) Match {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cur := t.root
	matched := make([]string, 0, len(layers))
	depth := 0

	for _, raw := range layers {
		d := normalizeDigest(raw)
		next, ok := cur.children[d]
		if !ok {
			break
		}
		cur = next
		matched = append(matched, d)
		depth++
	}
	return Match{
		Depth:       depth,
		Node:        cur,
		MatchedPath: matched,
		Images:      append([]ImageMeta(nil), cur.images...), // copy
	}
}

// HasImagePath returns true if an image with exactly the given chain exists.
func (t *Trie) HasImagePath(layers []string) bool {
	m := t.LongestPrefix(layers)
	return m.Depth == len(layers) && len(m.Images) > 0
}

func (t *Trie) GetSize() int {
	return int(atomic.LoadInt64(&t.size))
}
