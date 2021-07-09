package transformation

import (
	"bytes"
	"context"
	"encoding/base64"

	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/keys"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/set"
)

// OneToOne is a transformation that changes one key into another key.
type OneToOne func(ctx context.Context, keys []byte) []byte

// Then chains the input OneToOne function, outputting a new OneToOne function that combines the two.
func (oto OneToOne) Then(fn OneToOne) OneToOne {
	return func(ctx context.Context, key []byte) []byte {
		return fn(ctx, oto(ctx, key))
	}
}

// ThenMapToMany chains the input OneToMany function.
func (oto OneToOne) ThenMapToMany(fn OneToMany) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		return fn(ctx, oto(ctx, key))
	}
}

// ThenMapToBool takes in a OneToBool and returns a function which
// is a passed a context and key and determines if a key matches the check
func (oto OneToOne) ThenMapToBool(fn OneToBool) OneToBool {
	return func(ctx context.Context, key []byte) bool {
		return fn(ctx, oto(ctx, key))
	}
}

// MapEachToOne converts the input OneToOne function to a ManyToMany by applying it to all input keys one by one.
func MapEachToOne(fn OneToOne) ManyToMany {
	return func(ctx context.Context, keys [][]byte) [][]byte {
		for i, key := range keys {
			keys[i] = fn(ctx, key)
		}
		return keys
	}
}

// AddPrefix adds the given bucket prefix to the keys before output.
func AddPrefix(prefix []byte) OneToOne {
	dbPrefix := dbhelper.GetBucketKey(prefix, nil)
	return func(ctx context.Context, key []byte) []byte {
		ret := make([]byte, 0, len(key)+len(dbPrefix))
		ret = append(ret, dbPrefix...)
		ret = append(ret, key...)
		return ret
	}
}

// StripPrefixUnchecked removes the input bucket prefix from the keys before output. It must only
// be used if you can be absolutely sure that all keys will have the prefix.
func StripPrefixUnchecked(prefix []byte) OneToOne {
	prefixLen := dbhelper.GetBucketKeyLen(prefix)
	return func(ctx context.Context, key []byte) []byte {
		return key[prefixLen:]
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

// OneToBool is a function that takes in a key and returns a bool
type OneToBool func(ctx context.Context, key []byte) bool

// OneToMany is a transformation that changes one key into many new keys
type OneToMany func(ctx context.Context, key []byte) [][]byte

// Then chains the input ManyToMany function, applying it to all output keys at once.
func (otm OneToMany) Then(fn ManyToMany) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		return fn(ctx, otm(ctx, key))
	}
}

// ThenMapEachToBool takes in a OneToBool and returns a OneToBool based on the given input.
func (otm OneToMany) ThenMapEachToBool(fn OneToBool) OneToBool {
	return func(ctx context.Context, key []byte) bool {
		for _, k := range otm(ctx, key) {
			if fn(ctx, k) {
				return true
			}
		}
		return false
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
		// try to re-use whatever space is left over (e.g., because of a dedupe)
		ret := keys[len(keys):]
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
		seen := set.StringSet(make(map[string]struct{}, len(keys)))
		deduped := keys[:0]
		for _, key := range keys {
			if seen.Add(string(key)) {
				deduped = append(deduped, key)
			}
		}
		return deduped
	}
}

// ForwardFromContext steps forward (finding the values that are pointed to FROM the input keys) in the input RGraph for
// all the input keys, filtering on the given prefix.
func ForwardFromContext(prefix []byte) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		g := graph.GetGraph(ctx)
		if g == nil {
			return nil
		}
		return g.GetRefsFromPrefix(key, prefix)
	}
}

// BackwardFromContext steps backwards (finding the values that point TO the input keys) in the input graph for all the input keys.
func BackwardFromContext(prefix []byte) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		g := graph.GetGraph(ctx)
		if g == nil {
			return nil
		}
		return g.GetRefsToPrefix(key, prefix)
	}
}

// BackwardExistence returns if any key with the specified prefix exists.
func BackwardExistence(prefix []byte) func(ctx context.Context, key []byte) bool {
	return func(ctx context.Context, key []byte) bool {
		g := graph.GetGraph(ctx)
		if g == nil {
			return false
		}
		return g.ReferencedFromPrefix(key, prefix)
	}
}

// Many returns the results of all the given transformations.
func Many(fs ...OneToMany) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		var all [][]byte
		for _, f := range fs {
			all = append(all, f(ctx, key)...)
		}
		return all
	}
}

// ForwardEdgeKeys produces a group of pair keys that represent edges.
// The first OneToMany function produces the keys that become the first keys in the pair keys produced.
// The second transforms the first keys into a list of second keys, which will be used to create the edges.
// For example, if input is k:
// Step 1) toP(ctx, k) outputs { k1, k2 }
// Step 2) pToC(ctx, k1) outputs { c1, c2 }, and pToC(ctx, k2) outputs { c1, c3 }
// Final Output) { pairKey(k1, c1), pairKey(k1, c2), pairKey(k2, c1), pairKey(k2, c3) }
func ForwardEdgeKeys(toP, pToC OneToMany) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		ps := toP(ctx, key)
		ret := make([][]byte, 0, len(ps))
		for _, p := range ps {
			for _, c := range pToC(ctx, p) {
				ret = append(ret, keys.CreatePairKey(p, c))
			}
		}
		return ret
	}
}

// ReverseEdgeKeys works essentially the same way as ForwardEdgeKeys, however the output pair keys produced have the
// order of the keys reversed.
// Where ForwardEdgeKeys would produce
// Final Output) { pairKey(k1, c1), pairKey(k1, c2), pairKey(k2, c1), pairKey(k2, c3) }
// ReverseEdgeKeys would produce
// Final Output) { pairKey(c1, k1), pairKey(c2, k1), pairKey(c1, k2), pairKey(c3, k2) }
func ReverseEdgeKeys(toP, pToC OneToMany) OneToMany {
	return func(ctx context.Context, key []byte) [][]byte {
		ps := toP(ctx, key)
		ret := make([][]byte, 0, len(ps))
		for _, p := range ps {
			for _, c := range pToC(ctx, p) {
				ret = append(ret, keys.CreatePairKey(c, p))
			}
		}
		return ret
	}
}
