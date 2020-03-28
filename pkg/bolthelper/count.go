package bolthelper

import (
	bolt "github.com/etcd-io/bbolt"
)

// CountLeavesRecursive counts the number of all leaves in the given bucket and nested buckets up to the given max
// depth. The result is returned in an output parameter, since this function needs to be called inside a bolt
// transaction.
func CountLeavesRecursive(b *bolt.Bucket, maxDepth int, result *int) error {
	stats := b.Stats()
	numLeaves := stats.KeyN
	nestedBuckets := stats.BucketN - 1
	numLeaves -= nestedBuckets
	// Stats returns the number of *all* leaves across all nested buckets. This doesn't matter if we're querying with
	// infinite depth or if there are no nested buckets; otherwise, we need to count manually using `ForEach`.
	if maxDepth >= 0 || nestedBuckets > 0 {
		numLeaves = 0
		err := b.ForEach(func(k, v []byte) error {
			if v != nil {
				numLeaves++
				return nil
			}
			if maxDepth == 0 {
				return nil
			}
			return CountLeavesRecursive(b.Bucket(k), maxDepth-1, &numLeaves)
		})
		if err != nil {
			return err
		}
	}
	*result += numLeaves
	return nil
}
