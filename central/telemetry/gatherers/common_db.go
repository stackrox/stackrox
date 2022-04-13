package gatherers

import (
	"strings"

	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/stackrox/stackrox/pkg/telemetry/data"
)

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

func getBucketStats(prefixCardinality []*dto.Metric, prefixBytes []*dto.Metric) ([]*data.BucketStats, []error) {
	var errList []error
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
