package generic

import (
	"fmt"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/dbhelper"
	bolt "go.etcd.io/bbolt"
)

type crudImpl struct {
	bucketRef bolthelper.BucketRef

	deserializeFunc DeserializeFunc
	serializeFunc   SerializeFunc

	writeVersion uint64
}

func find(bucket *bolt.Bucket, firstKey Key, restKeys ...Key) (*bolt.Bucket, Key) {
	if len(restKeys) == 0 {
		return bucket, firstKey
	}

	currBucket := bucket
	for i := range restKeys {
		var key Key
		if i == 0 {
			key = firstKey
		} else {
			key = restKeys[i]
		}
		nextBucket := currBucket.Bucket(key)
		if nextBucket == nil {
			return nil, nil
		}
		currBucket = nextBucket
	}
	return currBucket, restKeys[len(restKeys)-1]
}

func traverse(bucket *bolt.Bucket, createBuckets bool, path KeyPath) (*bolt.Bucket, error) {
	currBucket := bucket
	for _, key := range path {
		nextBucket := currBucket.Bucket(key)
		if nextBucket == nil {
			if !createBuckets {
				return nil, nil
			}
			nextBucket, err := currBucket.CreateBucket(key)
			if err != nil {
				return nil, errors.Wrap(err, "creating bucket")
			}
			currBucket = nextBucket
		}
	}
	return currBucket, nil
}

func (c *crudImpl) incAndGetWriteVersion() uint64 {
	return atomic.AddUint64(&c.writeVersion, 1)
}

// Read reads and returns a single value from bolt.
func (c *crudImpl) Read(firstKey Key, restKeys ...Key) (interface{}, error) {
	var result interface{}
	err := c.bucketRef.View(func(b *bolt.Bucket) error {
		innermostBucket, leafKey := find(b, firstKey, restKeys...)
		if innermostBucket == nil {
			return nil
		}
		bytes := innermostBucket.Get(leafKey)
		if bytes == nil {
			return nil
		}
		var err error
		result, err = c.deserializeFunc(leafKey, bytes)
		return err
	})
	return result, err
}

// ReadBatch reads and returns a list of values for a list of key paths in the same order.
func (c *crudImpl) ReadBatch(keyPaths ...KeyPath) ([]interface{}, []int, error) {
	results := make([]interface{}, 0, len(keyPaths))
	var missingIndices []int
	err := c.bucketRef.View(func(b *bolt.Bucket) error {
		for i, path := range keyPaths {
			if len(path) == 0 {
				return errors.New("path must be non-empty")
			}
			innermostBucket, leafKey := find(b, path[0], path[1:]...)
			if innermostBucket == nil {
				missingIndices = append(missingIndices, i)
				continue
			}
			v := innermostBucket.Get(leafKey)
			if v == nil {
				missingIndices = append(missingIndices, i)
				continue
			}

			result, err := c.deserializeFunc(leafKey, v)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		return nil
	})
	return results, missingIndices, err
}

func (c *crudImpl) readAllRecursive(maxDepth int, b *bolt.Bucket, currPath KeyPath, result *[]Entry) error {
	return b.ForEach(func(k, v []byte) error {
		if v == nil {
			if maxDepth >= 0 && len(currPath) >= maxDepth {
				return nil
			}
			nextPath := make(KeyPath, len(currPath)+1)
			copy(nextPath, currPath)

			nextKey := make(Key, len(k))
			copy(nextKey, k)
			nextPath[len(currPath)] = nextKey
			return c.readAllRecursive(maxDepth, b, nextPath, result)
		}
		elem, err := c.deserializeFunc(k, v)
		if err != nil {
			return err
		}
		*result = append(*result, Entry{
			Nesting: currPath,
			Value:   elem,
		})
		return nil
	})
}

// ReadAll returns all of values stored in the bucket (and nested bucket, as long as the depth does not exceed
// maxDepth).
func (c *crudImpl) ReadAll(maxDepth int, keyPathPrefix ...Key) ([]Entry, error) {
	resultEntries := make([]Entry, 0)
	// Read all the byte arrays.
	err := c.bucketRef.View(func(b *bolt.Bucket) error {
		currBucket := b
		for _, prefixKey := range keyPathPrefix {
			currBucket = b.Bucket(prefixKey)
			if currBucket == nil {
				return nil
			}
		}
		return c.readAllRecursive(maxDepth, currBucket, nil, &resultEntries)
	})
	if err != nil {
		return nil, err
	}
	return resultEntries, nil
}

func (c *crudImpl) CountLeaves(maxDepth int, keyPathPrefix ...Key) (int, error) {
	numLeaves := 0
	err := c.bucketRef.View(func(b *bolt.Bucket) error {
		currBucket := b
		for _, prefixKey := range keyPathPrefix {
			currBucket = currBucket.Bucket(prefixKey)
			if currBucket == nil {
				return nil
			}
		}
		return bolthelper.CountLeavesRecursive(b, maxDepth, &numLeaves)
	})
	if err != nil {
		return 0, err
	}
	return numLeaves, nil
}

// Create creates a new entry in bolt for the input value .
// Returns an error if an entry with a matching key exists.
func (c *crudImpl) Create(x interface{}, nesting ...Key) error {
	key, bytes, err := c.serializeFunc(x)
	if err != nil {
		return err
	}

	return c.bucketRef.Update(func(b *bolt.Bucket) error {
		innermostBucket, err := traverse(b, true, nesting)
		if err != nil {
			return err
		}
		if innermostBucket.Get(key) != nil {
			return fmt.Errorf("entry with key %s already exists", key)
		}
		return innermostBucket.Put(key, bytes)
	})
}

// Create creates new entries in bolt for the input value.
// Returns an error if any entry with a matching key already exists.
func (c *crudImpl) CreateBatch(entries []Entry, nestingPrefix ...Key) error {
	serializedValues := make([]dbhelper.KV, len(entries))
	for i, entry := range entries {
		key, bytes, err := c.serializeFunc(entry.Value)
		if err != nil {
			return err
		}
		serializedValues[i] = dbhelper.KV{Key: key, Value: bytes}
	}

	return c.bucketRef.Update(func(b *bolt.Bucket) error {
		currBucket := b
		for _, prefixKey := range nestingPrefix {
			var err error
			currBucket, err = b.CreateBucketIfNotExists(prefixKey)
			if err != nil {
				return err
			}
		}
		for i, kv := range serializedValues {
			innermostBucket, err := traverse(currBucket, true, entries[i].Nesting)
			if err != nil {
				return err
			}
			if innermostBucket.Get(kv.Key) != nil {
				return fmt.Errorf("entry with key %s already exists", entries[i].Nesting)
			}
			if err := innermostBucket.Put(kv.Key, kv.Value); err != nil {
				return err
			}
		}
		return nil
	})
}

// Update updates a new entry in bolt for the input value.
// Returns an error if an entry with the same key does not already exist.
func (c *crudImpl) Update(x interface{}, nestingPrefix ...Key) (uint64, uint64, error) {
	key, bytes, err := c.serializeFunc(x)
	if err != nil {
		return 0, 0, err
	}

	var writeVersion uint64
	var attempts uint64
	return writeVersion, attempts, c.bucketRef.Update(func(b *bolt.Bucket) error {
		writeVersion = c.incAndGetWriteVersion()
		attempts++
		innermostBucket, err := traverse(b, false, nestingPrefix)
		if err != nil {
			return err
		}
		if innermostBucket == nil {
			return fmt.Errorf("bucket for key %v does not exist", nestingPrefix)
		}
		if innermostBucket.Get(key) == nil {
			return fmt.Errorf("value for key %v does not exist", key)
		}
		return innermostBucket.Put(key, bytes)
	})
}

// Update updates the entries in bolt for the input values.
// Returns an error if any input value does not have an existing entry.
func (c *crudImpl) UpdateBatch(entries []Entry, nestingPrefix ...Key) (uint64, uint64, error) {
	serializedValues := make([]dbhelper.KV, len(entries))
	for i, entry := range entries {
		key, bytes, err := c.serializeFunc(entry.Value)
		if err != nil {
			return 0, 0, err
		}
		serializedValues[i] = dbhelper.KV{Key: key, Value: bytes}
	}

	var writeVersion uint64
	var attempts uint64
	return writeVersion, attempts, c.bucketRef.Update(func(b *bolt.Bucket) error {
		writeVersion = c.incAndGetWriteVersion()
		attempts++
		currBucket := b
		for _, prefixKey := range nestingPrefix {
			var err error
			currBucket, err = b.CreateBucketIfNotExists(prefixKey)
			if err != nil {
				return err
			}
		}
		for i, kv := range serializedValues {
			innermostBucket, err := traverse(currBucket, false, entries[i].Nesting)
			if err != nil {
				return err
			}
			if innermostBucket == nil {
				return fmt.Errorf("bucket for key %v does not exist", entries[i].Nesting)
			}
			if innermostBucket.Get(kv.Key) == nil {
				return fmt.Errorf("entry with key %s does not exist", entries[i].Nesting)
			}
			if err := innermostBucket.Put(kv.Key, kv.Value); err != nil {
				return err
			}
		}
		return nil
	})
}

// Upsert upserts the input value into bolt whether or not an entry with the same key already exists.
func (c *crudImpl) Upsert(x interface{}, nesting ...Key) (uint64, uint64, error) {
	key, bytes, err := c.serializeFunc(x)
	if err != nil {
		return 0, 0, err
	}

	var writeVersion uint64
	var attempts uint64
	return writeVersion, attempts, c.bucketRef.Update(func(b *bolt.Bucket) error {
		writeVersion = c.incAndGetWriteVersion()
		attempts++
		innermostBucket, err := traverse(b, true, nesting)
		if err != nil {
			return err
		}
		return innermostBucket.Put(key, bytes)
	})
}

// Upsert upserts the input values into bolt whether or not entries with the same keys already exist.
func (c *crudImpl) UpsertBatch(entries []Entry, nestingPrefix ...Key) (uint64, uint64, error) {
	serializedValues := make([]dbhelper.KV, len(entries))
	for i, entry := range entries {
		key, bytes, err := c.serializeFunc(entry.Value)
		if err != nil {
			return 0, 0, err
		}
		serializedValues[i] = dbhelper.KV{Key: key, Value: bytes}
	}

	var writeVersion uint64
	var attempts uint64
	return writeVersion, attempts, c.bucketRef.Update(func(b *bolt.Bucket) error {
		writeVersion = c.incAndGetWriteVersion()
		attempts++
		currBucket := b
		for _, prefixKey := range nestingPrefix {
			var err error
			currBucket, err = b.CreateBucketIfNotExists(prefixKey)
			if err != nil {
				return err
			}
		}
		for i, kv := range serializedValues {
			innermostBucket, err := traverse(currBucket, true, entries[i].Nesting)
			if err != nil {
				return err
			}
			if err := innermostBucket.Put(kv.Key, kv.Value); err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete delete the input value in bolt whether or not an entry with the given key exists.
func (c *crudImpl) Delete(firstKey Key, restKeys ...Key) (uint64, uint64, error) {
	var writeVersion uint64
	var attempts uint64
	return writeVersion, attempts, c.bucketRef.Update(func(b *bolt.Bucket) error {
		writeVersion = c.incAndGetWriteVersion()
		attempts++
		innermostBucket, leafKey := find(b, firstKey, restKeys...)
		if innermostBucket == nil {
			return nil
		}
		return innermostBucket.Delete(leafKey)
	})
}

// DeleteBatch deletes the values associated with all of the input keys in bolt.
func (c *crudImpl) DeleteBatch(keyPaths ...KeyPath) (uint64, uint64, error) {
	var writeVersion uint64
	var attempts uint64
	return writeVersion, attempts, c.bucketRef.Update(func(b *bolt.Bucket) error {
		writeVersion = c.incAndGetWriteVersion()
		attempts++
		for _, keyPath := range keyPaths {
			if len(keyPath) == 0 {
				return errors.New("key path must not be empty")
			}
			innermostBucket, leafKey := find(b, keyPath[0], keyPath[1:]...)
			if innermostBucket == nil {
				return nil
			}
			if err := innermostBucket.Delete(leafKey); err != nil {
				return err
			}
		}
		return nil
	})
}
