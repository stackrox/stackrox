package dbhelper

// BucketHandler provides a few helper functions per-bucket prefix.
type BucketHandler struct {
	BucketPrefix []byte
}

// GetKey returns the prefixed key for the given id.
func (bh *BucketHandler) GetKey(id string) []byte {
	return GetBucketKey(bh.BucketPrefix, []byte(id))
}

// GetKeys returns the prefixed keys for the given ids.
func (bh *BucketHandler) GetKeys(ids ...string) [][]byte {
	keys := make([][]byte, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, bh.GetKey(id))
	}
	return keys
}

// GetID returns the ID for the prefixed key.
func (bh *BucketHandler) GetID(key []byte) string {
	return string(StripBucket(bh.BucketPrefix, key))
}

// GetIDs returns the ids for the prefixed keys.
func (bh *BucketHandler) GetIDs(keys ...[]byte) []string {
	ids := make([]string, 0, len(keys))
	for _, key := range keys {
		ids = append(ids, bh.GetID(key))
	}
	return ids
}

// Graph repliactes the DackBox graph interface, in order to avoid an import cycle.
type Graph interface {
	GetRefsFromPrefix(from, prefix []byte) [][]byte
	GetRefsToPrefix(to, prefix []byte) [][]byte

	CountRefsFromPrefix(from, prefix []byte) int
	CountRefsToPrefix(to, prefix []byte) int
}

// GetFilteredRefsFrom retrieves the refs from `from` in `g`, filtered to keys with this bucket's prefix.
func (bh *BucketHandler) GetFilteredRefsFrom(g Graph, from []byte) [][]byte {
	return g.GetRefsFromPrefix(from, bh.BucketPrefix)
}

// GetFilteredRefsTo retrieves the refs to `to` in `g`, filtered to keys with this bucket's prefix.
func (bh *BucketHandler) GetFilteredRefsTo(g Graph, to []byte) [][]byte {
	return g.GetRefsToPrefix(to, bh.BucketPrefix)
}

// CountFilteredRefsFrom counts the refs from `from` in `g`, filtered to keys with this bucket's prefix.
func (bh *BucketHandler) CountFilteredRefsFrom(g Graph, from []byte) int {
	return g.CountRefsFromPrefix(from, bh.BucketPrefix)
}

// CountFilteredRefsTo counts the refs to `to` in `g`, filtered to keys with this bucket's prefix.
func (bh *BucketHandler) CountFilteredRefsTo(g Graph, to []byte) int {
	return g.CountRefsToPrefix(to, bh.BucketPrefix)
}
