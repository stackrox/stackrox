package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const scannerBucket = "scanners"

// GetScanner returns scanner with given name.
func (b *BoltDB) GetScanner(name string) (scanner *v1.Scanner, exists bool, err error) {
	scanner = new(v1.Scanner)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(scannerBucket))
		val := b.Get([]byte(name))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, scanner)
	})

	return
}

// GetScanners retrieves scanners from bolt
func (b *BoltDB) GetScanners(request *v1.GetScannersRequest) ([]*v1.Scanner, error) {
	var scanners []*v1.Scanner
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(scannerBucket))
		b.ForEach(func(k, v []byte) error {
			var scanner v1.Scanner
			if err := proto.Unmarshal(v, &scanner); err != nil {
				return err
			}
			scanners = append(scanners, &scanner)
			return nil
		})
		return nil
	})
	return scanners, err
}

func (b *BoltDB) upsertScanner(scanner *v1.Scanner) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(scannerBucket))
		bytes, err := proto.Marshal(scanner)
		if err != nil {
			return err
		}
		err = b.Put([]byte(scanner.Name), bytes)
		return err
	})
}

// AddScanner upserts a scanner into bolt
func (b *BoltDB) AddScanner(scanner *v1.Scanner) error {
	return b.upsertScanner(scanner)
}

// UpdateScanner upserts a scanner into bolt
func (b *BoltDB) UpdateScanner(scanner *v1.Scanner) error {
	return b.upsertScanner(scanner)
}

// RemoveScanner removes a scanner from bolt
func (b *BoltDB) RemoveScanner(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(scannerBucket))
		err := b.Delete([]byte(name))
		return err
	})
}
