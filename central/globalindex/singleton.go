package globalindex

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	// DefaultBlevePath is the default path to Bleve's on-disk files
	DefaultBlevePath = "/var/lib/stackrox/scorch.bleve"
	// DefaultTmpBlevePath is the default path to Bleve's temporary on-disk files
	// This should only be used for indexes that are built on startup
	DefaultTmpBlevePath = "/tmp/scorch.bleve"
)

var (
	once sync.Once

	globalIndex    bleve.Index
	globalTmpIndex bleve.Index
)

func initialize() {
	var err error
	globalIndex, err = InitializeIndices(DefaultBlevePath, PersistedIndex)
	if err != nil {
		panic(err)
	}

	globalTmpIndex, err = InitializeIndices(DefaultTmpBlevePath, EphemeralIndex)
	if err != nil {
		panic(err)
	}
}

// GetGlobalIndex provides the global bleve index to use for indexing.
func GetGlobalIndex() bleve.Index {
	once.Do(initialize)

	return globalIndex
}

// GetGlobalTmpIndex is used for objects that are rebuilt every Central startup
func GetGlobalTmpIndex() bleve.Index {
	once.Do(initialize)
	return globalTmpIndex
}
