package transformation

import (
	"bytes"
	"context"
	"encoding/base64"
	"sort"

	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/utils/trie"
)

// OneToOne is a transformation that changes one key into another key.
type OneToOne func(ctx context.Context, keys []byte) []byte

// Then chains the input OneToOne function, outputting a new OneToOne function that combines the two.
func (otm OneToOne) Then(fn OneToOne) OneToOne {
	return func(ctx context.Context, key []byte) []byte {
		return fn(ctx, otm(ctx, key))
	}
}

// ThenMapToMany chains the input OneToMany function.
func (otm OneToOne) ThenMapToMany(fn OneToMany) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		return fn(ctx, otm(ctx, key))
	}
}

// MapEachToOne converts the input OneToOne function to a ManyToMany by applying it to all input keys one by one.
func MapEachToOne(fn OneToOne) ManyToMany {
	return func(ctx context.Context, keys [][]byte) [][]byte {
		ret := make([][]byte, 0, len(keys))
		for _, key := range keys {
			ret = append(ret, fn(ctx, key))
		}
		return ret
	}
}

// AddPrefix adds the given bucket prefix to the keys before output.
func AddPrefix(prefix []byte) OneToOne {
	return func(ctx context.Context, key []byte) []byte {
		return badgerhelper.GetBucketKey(prefix, key)
	}
}

// StripPrefix removes the input bucket prefix from the keys before output.
func StripPrefix(prefix []byte) OneToOne {
	return func(ctx context.Context, key []byte) []byte {
		return badgerhelper.StripBucket(prefix, key)
	}
}

// Split splits the input key into a set of output keys.
func Split(sep []byte) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		return bytes.Split(key, sep)
	}
}

// AtIndex outputs only the key at the input index.
func AtIndex(index int) ManyToMany {
	return func(ctx context.Context, keys [][]byte) [][]byte {
		if len(keys) < index {
			return nil
		}
		return [][]byte{keys[index]}
	}
}

// Decode applies RawURL decoding to the input key.
func Decode() OneToOne {
	return func(ctx context.Context, key []byte) []byte {
		if len(key) == 0 {
			return nil
		}
		ret := make([]byte, len(key))
		num, err := base64.RawURLEncoding.Decode(ret, key)
		if err != nil {
			return nil
		}
		return ret[:num]
	}
}

// OneToMany is a transformation that changes one key into many new keys
type OneToMany func(ctx context.Context, key []byte) [][]byte

// Then chains the input ManyToMany function, applying it to all output keys at once.
func (otm OneToMany) Then(fn ManyToMany) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		return fn(ctx, otm(ctx, key))
	}
}

// ThenMapEachToMany chains the input OneToMany function, applying it to each output key.
func (otm OneToMany) ThenMapEachToMany(fn OneToMany) OneToMany {
	outer := MapEachToMany(fn)
	return func(ctx context.Context, key []byte) [][]byte {
		return outer(ctx, otm(ctx, key))
	}
}

// ThenMapEachToOne chains the input OneToOne function, applying it to all output keys at once.
func (otm OneToMany) ThenMapEachToOne(fn OneToOne) OneToMany {
	outer := MapEachToOne(fn)
	return func(ctx context.Context, key []byte) [][]byte {
		return outer(ctx, otm(ctx, key))
	}
}

// MapEachToMany converts a OneToMany into a ManyToMany by applying it to all of the keys one by one.
func MapEachToMany(fn OneToMany) ManyToMany {
	return func(ctx context.Context, keys [][]byte) [][]byte {
		ret := make([][]byte, 0, len(keys))
		for _, key := range keys {
			ret = append(ret, fn(ctx, key)...)
		}
		return ret
	}
}

// ManyToMany is a transformation that changes many keys into many new keys
type ManyToMany func(ctx context.Context, keys [][]byte) [][]byte

// Then chains the input ManyToMany function, applying it to all output keys at once.
func (otm ManyToMany) Then(fn ManyToMany) ManyToMany {
	return func(ctx context.Context, keys [][]byte) [][]byte {
		return fn(ctx, otm(ctx, keys))
	}
}

// ThenMapEachToMany chains the input OneToMany function, applying it to all output keys.
func (otm ManyToMany) ThenMapEachToMany(fn OneToMany) ManyToMany {
	outer := MapEachToMany(fn)
	return func(ctx context.Context, keys [][]byte) [][]byte {
		return outer(ctx, otm(ctx, keys))
	}
}

// ThenMapEachToOne chains the input OneToOne function, applying it to all output keys.
func (otm ManyToMany) ThenMapEachToOne(fn OneToOne) ManyToMany {
	outer := MapEachToOne(fn)
	return func(ctx context.Context, keys [][]byte) [][]byte {
		return outer(ctx, otm(ctx, keys))
	}
}

// Dedupe removed duplicate key values before outputing.
func Dedupe() ManyToMany {
	return func(ctx context.Context, keys [][]byte) [][]byte {
		keySet := trie.New()
		deduped := keys[:0]
		for _, key := range keys {
			if keySet.Contains(key) {
				continue
			}
			deduped = append(deduped, key)
			keySet.Insert(key)
		}
		return deduped
	}
}

// Sort sorts the output keys.
func Sort() ManyToMany {
	return func(ctx context.Context, keys [][]byte) [][]byte {
		sort.SliceStable(keys, func(i, j int) bool {
			return bytes.Compare(keys[i], keys[j]) < 0
		})
		return keys
	}
}

// Predicate represents a boolean on a key.
type Predicate func(key []byte) bool

// Filtered filters the input with an input predicate.
func Filtered(pred Predicate) ManyToMany {
	return func(ctx context.Context, keys [][]byte) [][]byte {
		filtered := keys[:0]
		for _, key := range keys {
			if !pred(key) {
				continue
			}
			filtered = append(filtered, key)
		}
		return filtered
	}
}

// HasPrefix filters out items that do not have the matching bucket prefix.
func HasPrefix(prefix []byte) ManyToMany {
	return Filtered(func(key []byte) bool {
		return badgerhelper.HasPrefix(prefix, key)
	})
}

// Forward steps forward (finding the values that are pointed to FROM the input keys) in the input RGraph for all the
// input keys.
func Forward(graph graph.RGraph) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		return graph.GetRefsFrom(key)
	}
}

// Backward steps backwards (finding the values that point TO the input keys) in the input graph for all the input keys.
func Backward(graph graph.RGraph) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		return graph.GetRefsTo(key)
	}
}

// ForwardFromContext steps forward (finding the values that are pointed to FROM the input keys) in the input RGraph for
// all the input keys.
func ForwardFromContext() OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		g := graph.GetGraph(ctx)
		if g == nil {
			return nil
		}
		return g.GetRefsFrom(key)
	}
}

// BackwardFromContext steps backwards (finding the values that point TO the input keys) in the input graph for all the input keys.
func BackwardFromContext() OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		g := graph.GetGraph(ctx)
		if g == nil {
			return nil
		}
		return g.GetRefsTo(key)
	}
}
