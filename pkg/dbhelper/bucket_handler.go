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

// FilterKeys filters the deployment keys out of a list of keys.
func (bh *BucketHandler) FilterKeys(keys [][]byte) [][]byte {
	ret := make([][]byte, 0, len(keys))
	for _, key := range keys {
		if HasPrefix(bh.BucketPrefix, key) {
			ret = append(ret, key)
		}
	}
	return ret
}
