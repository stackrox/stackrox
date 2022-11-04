package types

import (
	"github.com/stackrox/scanner/database"
)

// MetadataEnricher defines functions used for enriching metadata.
type MetadataEnricher interface {
	Metadata() interface{}
	Summary() string
}

// AppendFunc is the type of a callback provided to an Appender.
type AppendFunc func(metadataKey string, metadata MetadataEnricher, severity database.Severity)

// Appender represents anything that can fetch vulnerability metadata and
// append it to a Vulnerability.
type Appender interface {
	// BuildCache loads metadata into memory such that it can be quickly accessed
	// for future calls to Append.
	BuildCache(dumpDir string) error

	// Append adds metadata to the given database.Vulnerability.
	Append(name string, subCVEs []string, callback AppendFunc) error

	// PurgeCache deallocates metadata from memory after all calls to Append are
	// finished.
	PurgeCache()

	// Name returns the name of the appender
	Name() string
}
