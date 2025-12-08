package scannerv4

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionEncode(t *testing.T) {
	tests := []struct {
		name           string
		indexerVersion string
		matcherVersion string
		want           string
	}{
		{
			name:           "indexer version",
			indexerVersion: "4.7.5",
			want:           "indexer=4.7.5",
		},
		{
			name:           "matcher version",
			matcherVersion: "4.8.3",
			want:           "matcher=4.8.3",
		},
		{
			name:           "indexer and matcher versions",
			indexerVersion: "4.7.5",
			matcherVersion: "4.8.3",
			want:           "indexer=4.7.5&matcher=4.8.3",
		},
		{
			name:           "indexer and matcher v4 versions",
			indexerVersion: "v4",
			matcherVersion: "v4",
			want:           "indexer=v4&matcher=v4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v Version
			v.Indexer = tt.indexerVersion
			v.Matcher = tt.matcherVersion

			got, err := v.Encode()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeVersion(t *testing.T) {
	tests := []struct {
		name           string
		encodedVersion string
		want           Version
	}{
		{
			name:           "valid indexer version",
			encodedVersion: "indexer=4.7.5",
			want:           Version{Indexer: "4.7.5"},
		},
		{
			name:           "matcher version",
			encodedVersion: "matcher=4.8.3",
			want:           Version{Matcher: "4.8.3"},
		},
		{
			name:           "indexer and matcher versions",
			encodedVersion: "indexer=4.7.5&matcher=4.8.3",
			want:           Version{Indexer: "4.7.5", Matcher: "4.8.3"},
		},
		{
			name:           "indexer and matcher v4 versions",
			encodedVersion: "indexer=v4&matcher=v4",
			want:           Version{Indexer: "v4", Matcher: "v4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeVersion(tt.encodedVersion)
			assert.NoError(t, err)
			assert.Equal(t, got, tt.want)
		})
	}
}
