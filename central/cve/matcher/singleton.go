package matcher

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	mapperStore "github.com/stackrox/rox/central/imagev2/mapper/datastore"
	nsDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once       sync.Once
	cveMatcher *CVEMatcher
)

func initialize() {
	var err error
	cveMatcher, err = NewCVEMatcher(clusterDataStore.Singleton(), nsDataStore.Singleton(), mapperStore.Singleton())
	utils.CrashOnError(err)
}

// Singleton returns singleton instance of CVEMatcher
func Singleton() *CVEMatcher {
	once.Do(initialize)
	return cveMatcher
}
