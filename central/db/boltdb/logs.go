package boltdb

import (
	"bytes"
	"encoding/binary"
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
	"github.com/boltdb/bolt"
)

const logsBucket = "logs"

// GetLogs returns all of the logs stored in the DB.
func (b *BoltDB) GetLogs() ([]string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetRange", "Logs")

	logs := make([]string, 0)
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(logsBucket)).Cursor()

		for k, v := bucket.First(); k != nil; k, v = bucket.Next() {
			logs = append(logs, string(v))
		}
		return nil
	})
	return logs, err
}

// CountLogs returns the number of logs in the DB.
func (b *BoltDB) CountLogs() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Count", "Logs")
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(logsBucket))
		count = b.Stats().KeyN
		return nil
	})
	return
}

// GetLogsRange returns the time range (inclusive) in unix seconds of all the logs stored.
func (b *BoltDB) GetLogsRange() (start int64, end int64, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetRange", "Logs")

	var min int64
	var max int64
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(logsBucket)).Cursor()

		minBytes, _ := bucket.First()
		min = int64(binary.LittleEndian.Uint64(minBytes))
		maxBytes, _ := bucket.Last()
		max = int64(binary.LittleEndian.Uint64(maxBytes))

		return nil
	})
	return min, max, err
}

// AddLog adds a log to bolt.
func (b *BoltDB) AddLog(log string) error {
	logTime := time.Now().Unix()

	defer metrics.SetBoltOperationDurationTime(time.Now(), "Add", "Logs")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(logsBucket))

		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(logTime))

		return bucket.Put(b, []byte(log))
	})
}

// RemoveLogs removes the logs in the specified time range (inclusive, unix seconds) from bolt.
func (b *BoltDB) RemoveLogs(from, to int64) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Remove", "Logs")

	errors := make([]error, 0)
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(logsBucket)).Cursor()

		startBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(startBytes, uint64(from))
		endBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(endBytes, uint64(to))

		for k, _ := bucket.Seek(startBytes); k != nil && bytes.Compare(k, endBytes) <= 0; k, _ = bucket.Next() {
			if err := bucket.Delete(); err != nil {
				errors = append(errors, err)
			}
		}

		if len(errors) == 0 {
			return nil
		}
		return errorhelpers.FormatErrors("errors deleting logs", errors)
	})
}
