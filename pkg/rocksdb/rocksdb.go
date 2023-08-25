package rocksdb

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/tecbot/gorocksdb"
	"go.uber.org/atomic"
)

// Make sure we link against the liblz4 compression library, as this is the compression algorithm
// we want to use.
// Note: older versions of github.com/tecbot/gorocksdb already add this to LDFLAGS, but more recent
// ones do not.

// #cgo LDFLAGS: -llz4
import "C"

// RocksDB is a wrapper around the base rocks DB, but implements synchronous close
type RocksDB struct {
	opsInProgress atomic.Uint32
	closing       atomic.Bool
	dir           string

	*gorocksdb.DB
}

var (
	log = logging.LoggerForModule()

	errDBClosed = errors.New("RocksDB is closed")
)

// IncRocksDBInProgressOps increments the wait group or returns an error if the DB is closed
func (r *RocksDB) IncRocksDBInProgressOps() error {
	if r.closing.Load() {
		return errDBClosed
	}
	r.opsInProgress.Inc()
	if r.closing.Load() {
		r.opsInProgress.Dec()
		return errDBClosed
	}
	return nil
}

// DecRocksDBInProgressOps removes the op from the in progress wait group
func (r *RocksDB) DecRocksDBInProgressOps() {
	r.opsInProgress.Dec()
}

// Close waits for all operations to complete and then closes the underlying RocksDB
func (r *RocksDB) Close() {
	log.Info("Signaling RocksDB to close")
	r.closing.Store(true)

	// Wait for all operations to finish
	startClose := time.Now()
	for ops := r.opsInProgress.Load(); ops != 0; ops = r.opsInProgress.Load() {
		log.Infof("Waiting for %d in progress RocksDB operations to complete", ops)
		time.Sleep(10 * time.Millisecond)

		if devbuild.IsEnabled() && time.Since(startClose) > 3*time.Second {
			panic("Closing RocksDB took too long more than 3s")
		}
	}
	log.Info("Closing RocksDB now that all operations have completed")

	// Close now that no operations are in progress
	if r.DB != nil {
		r.DB.Close()
	}
	log.Info("Closed RocksDB")
}

// NewTemp creates a new DB, but places it in the host temporary directory.
func NewTemp(name string) (*RocksDB, error) {
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("rocksdb-%s", strings.Replace(name, "/", "_", -1)))
	if err != nil {
		return nil, err
	}
	return New(tmpDir)
}

// CloseAndRemove closes the database and removes it. Should only be used for testing
func CloseAndRemove(db *RocksDB) error {
	db.Close()
	return os.RemoveAll(db.dir)
}

// New creates a new RocksDB at the specified path
func New(path string) (*RocksDB, error) {
	db, err := gorocksdb.OpenDb(GetRocksDBOptions(), path)
	if err != nil {
		return nil, err
	}
	return &RocksDB{
		DB:  db,
		dir: path,
	}, nil
}

// GetRocksDBOptions returns the options used to open RocksDB.
func GetRocksDBOptions() *gorocksdb.Options {
	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	// IMPORTANT: If you change the compression algorithm here, be sure to also change the #cgo
	// pragma directive in the import block to include the adequate library.
	opts.SetCompression(gorocksdb.LZ4Compression)
	opts.SetUseFsync(true)
	return opts
}
