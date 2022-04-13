package compact

import (
	"time"

	"github.com/stackrox/stackrox/migrator/log"
	bolt "go.etcd.io/bbolt"
)

const txMaxSize = 20 * 1024 * 1024

func compact(dst, src *bolt.DB, fillPercent float64) error {
	var totalKeysWritten int

	// commit regularly, or we'll run out of memory for large datasets if using one transaction.
	var size int64
	tx, err := dst.Begin(true)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != bolt.ErrTxClosed {
			log.WriteToStderrf("error rolling back: %v", err)
		}
	}()

	err = walk(src, func(keys [][]byte, k, v []byte, seq uint64) error {
		// On each key/value, check if we have exceeded tx size.
		sz := int64(len(k) + len(v))
		if size+sz > txMaxSize {
			// Commit previous transaction.
			t := time.Now()
			if err := tx.Commit(); err != nil {
				return err
			}
			log.WriteToStderrf("Took %0.2f seconds to commit", time.Since(t).Seconds())

			// Start new transaction.
			tx, err = dst.Begin(true)
			if err != nil {
				return err
			}
			size = 0
		}
		size += sz

		// Create bucket on the root transaction if this is the first level.
		nk := len(keys)
		if nk == 0 {
			bkt, err := tx.CreateBucket(k)
			if err != nil {
				return err
			}
			if err := bkt.SetSequence(seq); err != nil {
				return err
			}
			return nil
		}

		// Create buckets on subsequent levels, if necessary.
		b := tx.Bucket(keys[0])
		if nk > 1 {
			for _, k := range keys[1:] {
				b = b.Bucket(k)
			}
		}

		// Fill the entire page for best compaction.
		b.FillPercent = fillPercent

		// If there is no value then this is a bucket call.
		if v == nil {
			bkt, err := b.CreateBucket(k)
			if err != nil {
				return err
			}
			if err := bkt.SetSequence(seq); err != nil {
				return err
			}
			return nil
		}

		totalKeysWritten++
		if totalKeysWritten%5000 == 0 {
			log.WriteToStderrf("Compaction has written %d keys", totalKeysWritten)
		}
		// Otherwise treat it as a key/value pair.
		return b.Put(k, v)
	})
	if err != nil {
		return err
	}

	return tx.Commit()
}

// walkFunc is the type of the function called for keys (buckets and "normal"
// values) discovered by Walk. keys is the list of keys to descend to the bucket
// owning the discovered key/value pair k/v.
type walkFunc func(keys [][]byte, k, v []byte, seq uint64) error

// walk walks recursively the bolt database db, calling walkFn for each key it finds.
func walk(db *bolt.DB, walkFn walkFunc) error {
	return db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			return walkBucket(b, nil, name, nil, b.Sequence(), walkFn)
		})
	})
}

func walkBucket(b *bolt.Bucket, keypath [][]byte, k, v []byte, seq uint64, fn walkFunc) error {
	// Execute callback.
	if err := fn(keypath, k, v, seq); err != nil {
		return err
	}

	// If this is not a bucket then stop.
	if v != nil {
		return nil
	}

	// Iterate over each child key/value.
	keypath = append(keypath, k)
	return b.ForEach(func(k, v []byte) error {
		if v == nil {
			bkt := b.Bucket(k)
			return walkBucket(bkt, keypath, k, nil, bkt.Sequence(), fn)
		}
		return walkBucket(b, keypath, k, v, b.Sequence(), fn)
	})
}
