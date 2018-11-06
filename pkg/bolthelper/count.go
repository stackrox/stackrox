package bolthelper

import "github.com/boltdb/bolt"

// CountLeavesRecursive counts the number of all leaves in the given bucket and nested buckets up to the given max
// depth. The result is returned in an output parameter, since this function needs to be called inside a bolt
// transaction.
func CountLeavesRecursive(b *bolt.Bucket, maxDepth int, result *int) error {
	stats := b.Stats()
	numLeaves := stats.KeyN
	nestedBuckets := stats.BucketN - 1
	if nestedBuckets > 0 {
		numLeaves -= nestedBuckets
		if maxDepth != 0 {
			return b.ForEach(func(k, v []byte) error {
				if v != nil {
					return nil
				}
				nestedBucket := b.Bucket(k)
				return CountLeavesRecursive(nestedBucket, maxDepth-1, result)
			})
		}
	}
	*result += numLeaves
	return nil
}
