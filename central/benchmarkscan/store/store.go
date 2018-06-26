package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const (
	scanMetadataBucket      = "scan_metadata"
	benchmarksToScansBucket = "benchmarks_to_scans"
	checkResultsBucket      = "check_results"
	scansToCheckBucket      = "scans_to_checks"
)

// Store provides storage functionality for alerts.
type Store interface {
	AddScan(request *v1.BenchmarkScanMetadata) error
	ListBenchmarkScans(*v1.ListBenchmarkScansRequest) ([]*v1.BenchmarkScanMetadata, error)
	GetBenchmarkScan(request *v1.GetBenchmarkScanRequest) (*v1.BenchmarkScan, bool, error)
	GetHostResults(request *v1.GetHostResultsRequest) (*v1.HostResults, bool, error)
	AddBenchmarkResult(benchmark *v1.BenchmarkResult) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, scanMetadataBucket)
	bolthelper.RegisterBucket(db, benchmarksToScansBucket)
	bolthelper.RegisterBucket(db, checkResultsBucket)
	bolthelper.RegisterBucket(db, scansToCheckBucket)
	return &storeImpl{
		DB: db,
	}
}
