package client

import (
	"testing"

	"github.com/stackrox/rox/pkg/scannerv4"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestSetServiceVersion(t *testing.T) {
	tests := map[string]struct {
		version          *scannerv4.Version
		responseMetadata metadata.MD
		expectedVersion  string
	}{
		"version from metadata is used": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, "v4.8.0"),
			expectedVersion:  "v4.8.0",
		},
		"nil metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: nil,
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"empty metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.MD{},
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"header missing from metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs("other-header", "value"),
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"empty string version in metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, ""),
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"whitespace-only version in metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, "   "),
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"tab whitespace version in metadata uses default": {
			version:          &scannerv4.Version{},
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, "\t\n"),
			expectedVersion:  scannerv4.DefaultVersion,
		},
		"nil version is a no-op": {
			version:          nil,
			responseMetadata: metadata.Pairs(scannerv4.ServiceVersionHeader, "v4.8.0"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			options := callOptions{version: tt.version}
			setMatcherVersion(options, tt.responseMetadata)
			setIndexerVersion(options, tt.responseMetadata)

			if tt.version == nil {
				assert.Nil(t, options.version)
			} else {
				assert.Equal(t, tt.expectedVersion, tt.version.Matcher, "matcher version")
				assert.Equal(t, tt.expectedVersion, tt.version.Indexer, "indexer version")
			}
		})
	}
}
