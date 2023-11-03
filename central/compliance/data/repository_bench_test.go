package data

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/compliance/compress"
	"github.com/stackrox/rox/pkg/utils"
)

// To run this benchmark download sample data from the Compliance Checks In Nodes design doc

func BenchmarkUncompressResults(b *testing.B) {
	b.Skip("ROX-20480: This test is failing. Skipping!")
	complianceMap := map[string]*compliance.ComplianceReturn{
		"test": getCompressedCheckResults(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getNodeResults(complianceMap)
	}
}

func getCompressedCheckResults() *compliance.ComplianceReturn {
	uncompressedResults := readCheckResults()

	wrappedResults := compress.ResultWrapper{
		ResultMap: uncompressedResults,
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		panic(err)
	}

	defer utils.IgnoreError(gz.Close)
	if err := json.NewEncoder(gz).Encode(wrappedResults); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}
	compressedResults := &compliance.GZIPDataChunk{
		Gzip: buf.Bytes(),
	}

	return &compliance.ComplianceReturn{
		NodeName: "test",
		ScrapeId: "test scrape",
		Time:     types.TimestampNow(),
		Evidence: compressedResults,
	}
}

func readCheckResults() map[string]*compliance.ComplianceStandardResult {
	jsonFile, err := os.Open("repository_bench_test_data.json")
	if err != nil {
		panic(err)
	}
	defer utils.IgnoreError(jsonFile.Close)

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}

	var data map[string]*compliance.ComplianceStandardResult
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		panic(err)
	}
	return data
}
