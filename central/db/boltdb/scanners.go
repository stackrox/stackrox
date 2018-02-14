package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const scannerBucket = "scanners"

func (b *BoltDB) getScanner(id string, bucket *bolt.Bucket) (scanner *v1.Scanner, exists bool, err error) {
	scanner = new(v1.Scanner)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, scanner)
	return
}

// GetScanner returns scanner with given id.
func (b *BoltDB) GetScanner(id string) (scanner *v1.Scanner, exists bool, err error) {
	scanner = new(v1.Scanner)
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(scannerBucket))
		scanner, exists, err = b.getScanner(id, bucket)
		return err
	})
	return
}

// GetScanners retrieves scanners from bolt
func (b *BoltDB) GetScanners(request *v1.GetScannersRequest) ([]*v1.Scanner, error) {
	var scanners []*v1.Scanner
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(scannerBucket))
		return b.ForEach(func(k, v []byte) error {
			var scanner v1.Scanner
			if err := proto.Unmarshal(v, &scanner); err != nil {
				return err
			}
			scanners = append(scanners, &scanner)
			return nil
		})
	})
	return scanners, err
}

// AddScanner adds a scanner into bolt
func (b *BoltDB) AddScanner(scanner *v1.Scanner) (string, error) {
	scanner.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(scannerBucket))
		_, exists, err := b.getScanner(scanner.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Scanner %v (%v) cannot be added because it already exists", scanner.GetId(), scanner.GetName())
		}
		bytes, err := proto.Marshal(scanner)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(scanner.GetId()), bytes)
	})
	return scanner.Id, err
}

// UpdateScanner upserts a scanner into bolt
func (b *BoltDB) UpdateScanner(scanner *v1.Scanner) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(scannerBucket))
		bytes, err := proto.Marshal(scanner)
		if err != nil {
			return err
		}
		return b.Put([]byte(scanner.GetId()), bytes)
	})
}

// RemoveScanner removes a scanner from bolt
func (b *BoltDB) RemoveScanner(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(scannerBucket))
		key := []byte(id)
		if exists := b.Get(key) != nil; !exists {
			return db.ErrNotFound{Type: "Scanner", ID: string(key)}
		}
		return b.Delete(key)
	})
}
