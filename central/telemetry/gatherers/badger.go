package gatherers

import (
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type badgerGatherer struct {
	badger *badger.DB
}

func newBadgerGatherer(badger *badger.DB) *badgerGatherer {
	return &badgerGatherer{
		badger: badger,
	}
}

func getBadgerSize() (int64, error) {
	size, err := fileutils.DirectorySize(badgerhelper.DefaultBadgerPath)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func getName(metric *dto.Metric) string {
	for _, label := range metric.GetLabel() {
		if strings.ToLower(label.GetName()) == "prefix" {
			return label.GetValue()
		}
	}
	return "unknown"
}

func getGauge(metric *dto.Metric) (int64, error) {
	gauge := metric.GetGauge()
	if gauge == nil {
		return 0, errors.New("no metric found")
	}
	return int64(gauge.GetValue()), nil
}

func getSummedBucketStats(metrics []*dto.Metric) (map[string]int64, []error) {
	bucketStats := make(map[string]int64)
	var errorList []error
	for _, metric := range metrics {
		name := getName(metric)
		gaugeVal, err := getGauge(metric)
		if err != nil {
			errorList = append(errorList, errors.Wrapf(err, "getting metric for %s", name))
		}
		bucketStats[name] += gaugeVal
	}
	return bucketStats, errorList
}

func getOrCreateBucketStat(bucketName string, bucketStats map[string]*data.BucketStats) *data.BucketStats {
	bucket, ok := bucketStats[bucketName]
	if !ok {
		bucket = &data.BucketStats{
			Name: bucketName,
		}
		bucketStats[bucketName] = bucket
	}
	return bucket
}

// Gather returns telemetry information about the Badger database used by this central
func (d *badgerGatherer) Gather() *data.DatabaseStats {
	errorList := errorhelpers.NewErrorList("")
	sizeInBytes, err := getBadgerSize()
	errorList.AddError(err)

	bucketStats, bucketErrors := d.getBadgerBucketStats()
	errorList.AddErrors(bucketErrors...)

	dbStats := &data.DatabaseStats{
		Type: "badger",
		// Can't get the path from the DB object, we don't track the actual path.  Just use the default for now.
		Path:      badgerhelper.DefaultBadgerPath,
		UsedBytes: sizeInBytes,
		Buckets:   bucketStats,
		Errors:    errorList.ErrorStrings(),
	}
	return dbStats
}

func (d *badgerGatherer) getBadgerBucketStats() ([]*data.BucketStats, []error) {
	var errList []error
	prefixCardinality, prefixBytes, err := badgerhelper.GetBadgerMetrics()
	if err != nil {
		errList = append(errList, err)
	}

	if len(prefixCardinality) == 0 {
		return nil, nil
	}

	buckets := make(map[string]*data.BucketStats, len(prefixCardinality))
	summedCardinalities, cardErrors := getSummedBucketStats(prefixCardinality)
	errList = append(errList, cardErrors...)
	for bucketName, card := range summedCardinalities {
		bucket := getOrCreateBucketStat(bucketName, buckets)
		bucket.Cardinality = int(card)
	}

	summedBytes, byteErrors := getSummedBucketStats(prefixBytes)
	errList = append(errList, byteErrors...)
	for bucketName, sizeInBytes := range summedBytes {
		bucket := getOrCreateBucketStat(bucketName, buckets)
		bucket.UsedBytes = sizeInBytes
	}

	bucketSlice := make([]*data.BucketStats, 0, len(buckets))
	for _, bucket := range buckets {
		bucketSlice = append(bucketSlice, bucket)
	}
	return bucketSlice, errList
}
