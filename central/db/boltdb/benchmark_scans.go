package boltdb

import (
	"fmt"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const (
	scanMetadataBucket      = "scan_metadata"
	benchmarksToScansBucket = "benchmarks_to_scans"
	checkResultsBucket      = "check_results"
	scansToCheckBucket      = "scans_to_checks"
)

// AddScan inserts a scan into bolt
// It inserts data into two buckets.
// The first bucket is the benchmarksToScansBucket which is a mapping of benchmark identifier (currently Name) -> Scan Ids
// The second bucket is the scanMetadataBucket which is a mapping of scan IDs -> scan metadata
func (b *BoltDB) AddScan(request *v1.BenchmarkScanMetadata) error {
	return b.Update(func(tx *bolt.Tx) error {
		// Create benchmark bucket if does not already exist
		// Add scan id into that bucket
		topLevelBenchmarkBucket := tx.Bucket([]byte(benchmarksToScansBucket))
		benchmarkBucket, err := topLevelBenchmarkBucket.CreateBucketIfNotExists([]byte(request.GetBenchmark()))
		if err != nil {
			return err
		}
		// For now, add an empty object. It's just a mapping
		if err := benchmarkBucket.Put([]byte(request.GetScanId()), []byte{}); err != nil {
			return err
		}

		// Insert metadata into flat scan metadata bucket
		scanBucket := tx.Bucket([]byte(scanMetadataBucket))
		bytes, err := proto.Marshal(request)
		if err != nil {
			return err
		}
		return scanBucket.Put([]byte(request.GetScanId()), bytes)
	})
}

// AddBenchmarkResult adds a benchmark to bolt
// The schema of the addition is as follows:
// 1. scansToCheckBucket consists of ( scan id -> buckets based on top level check identifier (name for now, e.g. CIS 1.1). Inside that bucket is check result id -> empty
// 2. Flat check results bucket is a mapping of check result id -> check result
func (b *BoltDB) AddBenchmarkResult(result *v1.BenchmarkResult) error {
	return b.Update(func(tx *bolt.Tx) error {
		// iterate over all checks and add them into buckets with key (Name)
		scansToCheck := tx.Bucket([]byte(scansToCheckBucket))
		// Create scan id bucket if it doesn't exist
		scanIDBucket, err := scansToCheck.CreateBucketIfNotExists([]byte(result.GetScanId()))
		if err != nil {
			return err
		}

		checksBucket := tx.Bucket([]byte(checkResultsBucket))
		for _, check := range result.Results {
			check.Id = uuid.NewV4().String()
			check.ClusterId = result.GetClusterId()
			check.Host = result.GetHost()

			bytes, err := proto.Marshal(check)
			if err != nil {
				return err
			}
			// Add the check into the flat checkResultsBucket
			if err := checksBucket.Put([]byte(check.GetId()), bytes); err != nil {
				return err
			}
			// Create the top level check identifier for this current scan. e.g. UUID -> CIS 1.1
			specificCheckBucket, err := scanIDBucket.CreateBucketIfNotExists([]byte(check.GetDefinition().GetName()))
			if err != nil {
				return err
			}
			// Correlate the current check name to the check result id
			if err := specificCheckBucket.Put([]byte(check.GetId()), []byte{}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *BoltDB) getScanMetadata(tx *bolt.Tx, scanID string) (*v1.BenchmarkScanMetadata, error) {
	metadataBucket := tx.Bucket([]byte(scanMetadataBucket))
	bytes := metadataBucket.Get([]byte(scanID))
	if bytes == nil {
		return nil, db.ErrNotFound{Type: "Scan", ID: scanID}
	}
	var result v1.BenchmarkScanMetadata
	if err := proto.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *BoltDB) getCheckResult(tx *bolt.Tx, k []byte) (*v1.CheckResult, error) {
	checkBucket := tx.Bucket([]byte(checkResultsBucket))
	bytes := checkBucket.Get(k)
	var result v1.CheckResult
	if err := proto.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *BoltDB) fillScanCheck(check *v1.BenchmarkScan_Check, result *v1.CheckResult) {
	check.Definition = result.GetDefinition()
	check.AggregatedResults[result.GetResult().String()]++
	check.HostResults = append(check.GetHostResults(), &v1.BenchmarkScan_Check_HostResult{
		Host:   result.GetHost(),
		Result: result.GetResult(),
		Notes:  result.GetNotes(),
	})
}

// GetBenchmarkScan retrieves a scan from the database
func (b *BoltDB) GetBenchmarkScan(request *v1.GetBenchmarkScanRequest) (scan *v1.BenchmarkScan, exists bool, err error) {
	if request.GetScanId() == "" {
		err = fmt.Errorf("Scan id must be defined when retrieving results")
		return
	}
	clusterSet := newStringSet(request.GetClusterIds())
	hostSet := newStringSet(request.GetHosts())
	scan = new(v1.BenchmarkScan)
	err = b.View(func(tx *bolt.Tx) error {
		metadata, err := b.getScanMetadata(tx, request.GetScanId())
		if err != nil {
			return err
		}
		exists = true

		// grab from scan ids -> checks -> check ids
		scanToChecks := tx.Bucket([]byte(scansToCheckBucket)).Bucket([]byte(request.GetScanId()))
		if scanToChecks == nil {
			return db.ErrNotFound{Type: "Results for scan", ID: request.GetScanId()}
		}

		scan.Checks = make([]*v1.BenchmarkScan_Check, 0, len(metadata.GetChecks()))
		for _, check := range metadata.GetChecks() {
			// Initialize aggregated results
			scanCheck := &v1.BenchmarkScan_Check{
				AggregatedResults: make(map[string]int32),
			}

			resultBucket := scanToChecks.Bucket([]byte(check))
			if resultBucket == nil {
				return db.ErrNotFound{Type: "Results for check", ID: check}
			}

			// Iterate over the checks that are included in the desired scan and fetch them
			err = resultBucket.ForEach(func(k, v []byte) error {
				result, err := b.getCheckResult(tx, k)
				if err != nil {
					return err
				}
				if clusterSet.Cardinality() != 0 && !clusterSet.Contains(result.GetClusterId()) {
					return nil
				}
				if hostSet.Cardinality() != 0 && !hostSet.Contains(result.GetHost()) {
					return nil
				}
				b.fillScanCheck(scanCheck, result)
				return nil
			})
			if err != nil {
				return err
			}
			sort.SliceStable(scanCheck.HostResults, func(i, j int) bool { return scanCheck.HostResults[i].GetHost() < scanCheck.HostResults[j].GetHost() })
			scan.Checks = append(scan.GetChecks(), scanCheck)
		}
		return nil
	})
	return
}

// ListBenchmarkScans filters the scans by the request parameters
func (b *BoltDB) ListBenchmarkScans(request *v1.ListBenchmarkScansRequest) ([]*v1.BenchmarkScanMetadata, error) {
	var scansMetadata []*v1.BenchmarkScanMetadata
	err := b.View(func(tx *bolt.Tx) error {
		scanBucket := tx.Bucket([]byte(scanMetadataBucket))
		err := scanBucket.ForEach(func(k, v []byte) error {
			var metadata v1.BenchmarkScanMetadata
			if err := proto.Unmarshal(v, &metadata); err != nil {
				return err
			}
			scansMetadata = append(scansMetadata, &metadata)
			return nil
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	// Filter the schedule metadata
	clusterSet := newStringSet(request.GetClusterIds())
	filtered := scansMetadata[:0]
	for _, scan := range scansMetadata {
		if request.GetBenchmark() != "" && request.GetBenchmark() != scan.GetBenchmark() {
			continue
		}
		scanClusterSet := newStringSet(request.GetClusterIds())
		// This means none of the items intersect in the two clusters so we should skip this scan
		if clusterSet.Cardinality() != 0 && clusterSet.Intersect(scanClusterSet).Cardinality() == 0 {
			continue
		}
		filtered = append(filtered, scan)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return protoconv.CompareProtoTimestamps(filtered[i].GetTime(), filtered[j].GetTime()) == 1
	})
	return filtered, nil
}
