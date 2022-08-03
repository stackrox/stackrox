package wal

import (
	"encoding/binary"

	"github.com/cockroachdb/pebble"
)

type WAL interface {
	Insert(id string, hash uint64) error
	Delete(id string) error
	GetMap() map[string]uint64
}

func Open() WAL {
	db, err := pebble.Open("/tmp/wal", &pebble.Options{})
	if err != nil {
		log.Errorf("could not open WAL for Sensor. This could have adverse performance impacts: %v", err)
	}
	return &walImpl{
		db: db,
	}
}

type Entry struct {
	ID   string
	Hash int64
}

type walImpl struct {
	db *pebble.DB
}

func (w *walImpl) Insert(id string, hash uint64) error {
	if w.db == nil {
		return nil
	}
	hashBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(hashBytes, hash)
	return w.db.Set([]byte(id), hashBytes, &pebble.WriteOptions{
		Sync: true,
	})
}

func (w *walImpl) Delete(id string) error {
	if w.db == nil {
		return nil
	}
	return w.db.Delete([]byte(id), &pebble.WriteOptions{
		Sync: true,
	})
}

func (w *walImpl) GetMap() map[string]uint64 {
	hashes := make(map[string]uint64)

	if w.db == nil {
		return hashes
	}

	it := w.db.NewIter(&pebble.IterOptions{})
	defer it.Close()

	for it.Prev(); it.Valid(); it.Next() {
		hash := binary.LittleEndian.Uint64(it.Value())
		hashes[string(it.Key())] = hash
	}
	return hashes
}
