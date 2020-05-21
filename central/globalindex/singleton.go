package globalindex

import (
	"path/filepath"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	// DefaultBlevePath is the default path to Bleve's on-disk files
	DefaultBlevePath = "/var/lib/stackrox/scorch.bleve"
	// DefaultTmpBlevePath is the default path to Bleve's temporary on-disk files
	// This should only be used for indexes that are built on startup
	DefaultTmpBlevePath = "/tmp/scorch.bleve"

	// SeparateIndexPath returns path prefix for indexes that are going to be shareded into separate directories
	SeparateIndexPath = "/var/lib/stackrox/index"
)

var (
	once sync.Once

	globalIndex    bleve.Index
	globalTmpIndex bleve.Index

	separates     = make(map[string]bleve.Index)
	separatesLock sync.Mutex
)

func initialize() {
	var err error
	globalIndex, err = InitializeIndices("combined-persisted", DefaultBlevePath, PersistedIndex)
	if err != nil {
		panic(err)
	}

	globalTmpIndex, err = InitializeIndices("combined-ephemeral", DefaultTmpBlevePath, EphemeralIndex)
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

// GetAlertIndex returns the alert index on a separate index path
func GetAlertIndex() bleve.Index {
	return getSeparateIndex("alert")
}

// GetPodIndex returns the pod index in a separate index
func GetPodIndex() bleve.Index {
	return getSeparateIndex("pod")
}

// GetProcessIndex returns the process index in a separate index
func GetProcessIndex() bleve.Index {
	return getSeparateIndex("process")
}

func getSeparateIndex(obj string) bleve.Index {
	separatesLock.Lock()
	defer separatesLock.Unlock()
	if index, ok := separates[obj]; ok {
		return index
	}
	path := filepath.Join(SeparateIndexPath, obj)
	index, err := InitializeIndices(obj, path, PersistedIndex)
	if err != nil {
		panic(err)
	}
	separates[obj] = index
	return index
}
