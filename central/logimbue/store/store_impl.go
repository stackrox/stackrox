package store

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	ops "github.com/stackrox/stackrox/pkg/metrics"
	bolt "go.etcd.io/bbolt"
)

type storeImpl struct {
	*bolt.DB
}

// GetLogs returns all of the logs stored in the DB.
func (b *storeImpl) GetLogs() ([]string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "Logs")

	logs := make([]string, 0)
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(logsBucket).Cursor()

		for k, v := bucket.First(); k != nil; k, v = bucket.Next() {
			logs = append(logs, string(v))
		}
		return nil
	})
	return logs, err
}

// GetLogsRange returns the time range (inclusive) in unix seconds of all the logs stored.
func (b *storeImpl) GetLogsRange() (start int64, end int64, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Logs")

	var min int64
	var max int64
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(logsBucket).Cursor()

		minBytes, _ := bucket.First()
		min = int64(binary.LittleEndian.Uint64(minBytes))
		maxBytes, _ := bucket.Last()
		max = int64(binary.LittleEndian.Uint64(maxBytes))

		return nil
	})
	return min, max, err
}

// AddLog adds a log to bolt.
func (b *storeImpl) AddLog(log string) error {
	logTime := time.Now().Unix()

	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Logs")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(logsBucket)

		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(logTime))

		return bucket.Put(b, []byte(log))
	})
}

// RemoveLogs removes the logs in the specified time range (inclusive, unix seconds) from bolt.
func (b *storeImpl) RemoveLogs(from, to int64) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Logs")

	errorList := errorhelpers.NewErrorList("errors deleting logs")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(logsBucket).Cursor()

		startBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(startBytes, uint64(from))
		endBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(endBytes, uint64(to))

		for k, _ := bucket.Seek(startBytes); k != nil && bytes.Compare(k, endBytes) <= 0; k, _ = bucket.Next() {
			errorList.AddError(bucket.Delete())
		}
		return errorList.ToError()
	})
}
