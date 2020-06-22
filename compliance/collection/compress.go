package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	internalTypes "github.com/stackrox/rox/pkg/docker/types"
	"github.com/stackrox/rox/pkg/utils"
)

func compressDockerData(dockerData *internalTypes.Data) (*compliance.GZIPDataChunk, error) {
	return compress(dockerData)
}

func compressResults(results map[string]*compliance.ComplianceStandardResult) (*compliance.GZIPDataChunk, error) {
	return compress(results)
}

func compress(compressable interface{}) (*compliance.GZIPDataChunk, error) {
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
	return &compliance.GZIPDataChunk{
		Gzip: buf.Bytes(),
	}, nil
}
