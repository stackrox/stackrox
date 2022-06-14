package matcher

import (
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	imageDataStore "github.com/stackrox/stackrox/central/image/datastore"
	nsDataStore "github.com/stackrox/stackrox/central/namespace/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once       sync.Once
	cveMatcher *CVEMatcher
)

func initialize() {
	var err error
	cveMatcher, err = NewCVEMatcher(clusterDataStore.Singleton(), nsDataStore.Singleton(), imageDataStore.Singleton())
	utils.CrashOnError(err)
}

// Singleton returns singleton instance of CVEMatcher
func Singleton() *CVEMatcher {
	once.Do(initialize)
	return cveMatcher
}
