package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const (
	scanMetadataBucket      = "scan_metadata"
	benchmarksToScansBucket = "benchmarks_to_scans"
	checkResultsBucket      = "check_results"
	scansToCheckBucket      = "scans_to_checks"
)

// Store provides storage functionality for alerts.
type Store interface {
	AddScan(request *storage.BenchmarkScanMetadata) error
	ListBenchmarkScans(*v1.ListBenchmarkScansRequest) ([]*storage.BenchmarkScanMetadata, error)
	GetBenchmarkScan(request *v1.GetBenchmarkScanRequest) (*storage.BenchmarkScan, bool, error)
	GetHostResults(request *v1.GetHostResultsRequest) (*v1.HostResults, bool, error)
	AddBenchmarkResult(benchmark *storage.BenchmarkResult) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, scanMetadataBucket)
	bolthelper.RegisterBucketOrPanic(db, benchmarksToScansBucket)
	bolthelper.RegisterBucketOrPanic(db, checkResultsBucket)
	bolthelper.RegisterBucketOrPanic(db, scansToCheckBucket)
	return &storeImpl{
		DB: db,
	}
}
