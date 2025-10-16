package data

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"maps"
	"slices"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/compress"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/suite"
)

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

type RepositoryTestSuite struct {
	suite.Suite
}

func compressTestData(toCompress map[string]*compliance.ComplianceStandardResult) (*compliance.GZIPDataChunk, error) {
	compressable := &compress.ResultWrapper{
		ResultMap: toCompress,
	}
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	// It is safe to double-close the gzip writer and both closes are necessary.  Closing the writer flushes data and
	// writes gzip footers, so if there is an error closing the writer the zipped data will not be valid and we want to
	// return nil results.  However, we want to make sure close is always called even if there is an error in Encode()
	// so we want to defer the close as well.
	defer utils.IgnoreError(gz.Close)
	if err := json.NewEncoder(gz).Encode(compressable); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	gzipdc := &compliance.GZIPDataChunk{}
	if x := buf.Bytes(); x != nil {
		gzipdc.SetGzip(x)
	}
	return gzipdc, nil
}

func (s *RepositoryTestSuite) TestGetNodeResults() {
	testNodeName := "testNodeName"

	testEvidence := map[string]*compliance.ComplianceStandardResult{
		"testStandardName": compliance.ComplianceStandardResult_builder{
			NodeCheckResults: map[string]*storage.ComplianceResultValue{
				"testCheckName": storage.ComplianceResultValue_builder{
					Evidence: []*storage.ComplianceResultValue_Evidence{
						storage.ComplianceResultValue_Evidence_builder{
							State:   0,
							Message: "Joseph Rules",
						}.Build(),
					},
					OverallState: 0,
				}.Build(),
			},
		}.Build(),
	}

	compressedEvidence, err := compressTestData(testEvidence)
	s.Require().NoError(err)

	testScrapeResults := map[string]*compliance.ComplianceReturn{
		testNodeName: compliance.ComplianceReturn_builder{
			Evidence: compressedEvidence,
		}.Build(),
		"notDecompressable": compliance.ComplianceReturn_builder{
			Evidence: compliance.GZIPDataChunk_builder{
				Gzip: []byte("Not Decompressable"),
			}.Build(),
		}.Build(),
		"noEvidence": {},
	}

	nodeResults := getNodeResults(testScrapeResults)

	s.Equal(slices.Collect(maps.Keys(nodeResults)), []string{testNodeName})
	actual := nodeResults[testNodeName]
	protoassert.MapEqual(s.T(), testEvidence, actual)
}
