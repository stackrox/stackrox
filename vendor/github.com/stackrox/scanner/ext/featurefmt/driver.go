// Copyright 2017 clair authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package featurefmt exposes functions to dynamically register methods for
// determining the features present in an image layer.
package featurefmt

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/pkg/analyzer"
	rhelv2 "github.com/stackrox/scanner/pkg/rhelv2/rpm"
	"github.com/stackrox/scanner/pkg/tarutil"
	"github.com/stretchr/testify/assert"
)

var (
	listersM sync.RWMutex
	listers  = make(map[string]Lister)
)

// PackageKey is a key identifying a unique package.
type PackageKey struct {
	Name    string
	Version string
}

// Lister represents an ability to list the features present in an image layer.
type Lister interface {
	// ListFeatures produces a list of FeatureVersions present in an image layer.
	ListFeatures(analyzer.Files) ([]database.FeatureVersion, error)

	// RequiredFilenames returns the list of files required to be in the LayerFiles
	// provided to the ListFeatures method.
	//
	// Filenames must not begin with "/".
	RequiredFilenames() []string
}

// RegisterLister makes a Lister available by the provided name.
//
// If called twice with the same name, the name is blank, or if the provided
// Lister is nil, this function panics.
func RegisterLister(name string, l Lister) {
	if name == "" {
		panic("featurefmt: could not register a Lister with an empty name")
	}
	if l == nil {
		panic("featurefmt: could not register a nil Lister")
	}

	listersM.Lock()
	defer listersM.Unlock()

	if _, dup := listers[name]; dup {
		panic("featurefmt: RegisterLister called twice for " + name)
	}

	listers[name] = l
}

// ListFeatures produces the list of FeatureVersions in an image layer using
// every registered Lister.
func ListFeatures(files analyzer.Files) ([]database.FeatureVersion, error) {
	listersM.RLock()
	defer listersM.RUnlock()

	var totalFeatures []database.FeatureVersion
	for _, lister := range listers {
		features, err := lister.ListFeatures(files)
		if err != nil {
			return []database.FeatureVersion{}, err
		}
		totalFeatures = append(totalFeatures, features...)
	}

	return totalFeatures, nil
}

// RequiredFilenames returns the total list of files required for all
// registered Listers.
func RequiredFilenames() (files []string) {
	listersM.RLock()
	defer listersM.RUnlock()

	for _, lister := range listers {
		files = append(files, lister.RequiredFilenames()...)
	}

	files = append(files, rhelv2.RequiredFilenames()...)

	return
}

// TestData represents the data used to test an implementation of Lister.
type TestData struct {
	Files           tarutil.LayerFiles
	FeatureVersions []database.FeatureVersion
}

// LoadFileForTest can be used in order to obtain the []byte contents of a file
// that is meant to be used for test data.
func LoadFileForTest(name string) []byte {
	_, filename, _, _ := runtime.Caller(0)
	d, _ := os.ReadFile(filepath.Join(filepath.Dir(filename)) + "/" + name)
	return d
}

// TestLister runs a Lister on each provided instance of TestData and asserts
// the output to be equal to the expected output.
func TestLister(t *testing.T, l Lister, testData []TestData) {
	for _, td := range testData {
		featureVersions, err := l.ListFeatures(td.Files)
		if assert.Nil(t, err) && assert.Len(t, featureVersions, len(td.FeatureVersions)) {
			for _, expectedFeatureVersion := range td.FeatureVersions {
				assert.Contains(t, featureVersions, expectedFeatureVersion)
			}
		}
	}
}

// AddToDependencyMap checks and adds files to executable and library dependency
func AddToDependencyMap(filename string, fileData analyzer.FileData, execToDeps, libToDeps database.StringToStringsMap) {
	if fileData.Executable {
		deps := set.NewStringSet()
		if elfMeta := fileData.ELFMetadata; elfMeta != nil {
			deps.AddAll(elfMeta.ImportedLibraries...)
		}
		execToDeps[filename] = deps
	}
	if fileData.ELFMetadata != nil {
		for _, soname := range fileData.ELFMetadata.Sonames {
			deps, ok := libToDeps[soname]
			if !ok {
				deps = set.NewStringSet()
				libToDeps[soname] = deps
			}
			deps.AddAll(fileData.ELFMetadata.ImportedLibraries...)
		}
	}
}
