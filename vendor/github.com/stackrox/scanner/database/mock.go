// Copyright 2015 clair authors
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

//nolint:revive
package database

import (
	"time"

	"github.com/stackrox/scanner/pkg/component"
)

var _ Datastore = (*MockDatastore)(nil)

// MockDatastore implements Datastore and enables overriding each available method.
// The default behavior of each method is to simply panic.
type MockDatastore struct {
	FctListNamespaces                      func() ([]Namespace, error)
	FctInsertLayer                         func(Layer, string, *DatastoreOptions) error
	FctFindLayer                           func(name, lineage string, opts *DatastoreOptions) (Layer, error)
	FctDeleteLayer                         func(name string) error
	FctInsertRHELv2Layer                   func(*RHELv2Layer) error
	FctGetRHELv2Layers                     func(layer string) ([]*RHELv2Layer, error)
	FctGetRHELv2Vulnerabilities            func(records []*RHELv2Record) (map[int][]*RHELv2Vulnerability, error)
	FctListVulnerabilities                 func(namespaceName string, limit int, page int) ([]Vulnerability, int, error)
	FctInsertVulnerabilities               func(vulnerabilities []Vulnerability) error
	FctInsertRHELv2Vulnerabilities         func(vulnerabilities []*RHELv2Vulnerability) error
	FctFindVulnerability                   func(namespaceName, name string) (Vulnerability, error)
	FctDeleteVulnerability                 func(namespaceName, name string) error
	FctInsertVulnerabilityFixes            func(vulnerabilityNamespace, vulnerabilityName string, fixes []FeatureVersion) error
	FctDeleteVulnerabilityFix              func(vulnerabilityNamespace, vulnerabilityName, featureName string) error
	FctInsertKeyValue                      func(key, value string) error
	FctGetKeyValue                         func(key string) (string, error)
	FctLock                                func(name string, owner string, duration time.Duration, renew bool) (bool, time.Time)
	FctUnlock                              func(name, owner string)
	FctFindLock                            func(name string) (string, time.Time, error)
	FctPing                                func() bool
	FctClose                               func()
	FctInsertLayerComponents               func(l, lineage string, c []*component.Component, r []string, opts *DatastoreOptions) error
	FctGetLayerLanguageComponents          func(layer, lineage string, opts *DatastoreOptions) ([]*component.LayerToComponents, error)
	FctGetVulnerabilitiesForFeatureVersion func(featureVersion FeatureVersion) ([]Vulnerability, error)
	FctLoadVulnerabilities                 func(featureVersions []FeatureVersion) error
	FctFeatureExists                       func(namespace, feature string) (bool, error)
}

func (mds *MockDatastore) InsertLayer(layer Layer, lineage string, opts *DatastoreOptions) error {
	if mds.FctInsertLayer != nil {
		return mds.FctInsertLayer(layer, lineage, opts)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) FindLayer(name, lineage string, opts *DatastoreOptions) (Layer, error) {
	if mds.FctFindLayer != nil {
		return mds.FctFindLayer(name, lineage, opts)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) DeleteLayer(name string) error {
	if mds.FctDeleteLayer != nil {
		return mds.FctDeleteLayer(name)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) InsertRHELv2Layer(layer *RHELv2Layer) error {
	if mds.FctInsertRHELv2Layer != nil {
		return mds.FctInsertRHELv2Layer(layer)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) GetRHELv2Layers(layer string) ([]*RHELv2Layer, error) {
	if mds.FctGetRHELv2Layers != nil {
		return mds.FctGetRHELv2Layers(layer)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) GetRHELv2Vulnerabilities(records []*RHELv2Record) (map[int][]*RHELv2Vulnerability, error) {
	if mds.FctGetRHELv2Vulnerabilities != nil {
		return mds.FctGetRHELv2Vulnerabilities(records)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) ListVulnerabilities(namespaceName string, limit int, page int) ([]Vulnerability, int, error) {
	if mds.FctListVulnerabilities != nil {
		return mds.FctListVulnerabilities(namespaceName, limit, page)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) InsertVulnerabilities(vulnerabilities []Vulnerability) error {
	if mds.FctInsertVulnerabilities != nil {
		return mds.FctInsertVulnerabilities(vulnerabilities)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) InsertRHELv2Vulnerabilities(vulnerabilities []*RHELv2Vulnerability) error {
	if mds.FctInsertRHELv2Vulnerabilities != nil {
		return mds.FctInsertRHELv2Vulnerabilities(vulnerabilities)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) FindVulnerability(namespaceName, name string) (Vulnerability, error) {
	if mds.FctFindVulnerability != nil {
		return mds.FctFindVulnerability(namespaceName, name)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) DeleteVulnerability(namespaceName, name string) error {
	if mds.FctDeleteVulnerability != nil {
		return mds.FctDeleteVulnerability(namespaceName, name)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) InsertVulnerabilityFixes(vulnerabilityNamespace, vulnerabilityName string, fixes []FeatureVersion) error {
	if mds.FctInsertVulnerabilityFixes != nil {
		return mds.FctInsertVulnerabilityFixes(vulnerabilityNamespace, vulnerabilityName, fixes)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) DeleteVulnerabilityFix(vulnerabilityNamespace, vulnerabilityName, featureName string) error {
	if mds.FctDeleteVulnerabilityFix != nil {
		return mds.FctDeleteVulnerabilityFix(vulnerabilityNamespace, vulnerabilityName, featureName)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) InsertKeyValue(key, value string) error {
	if mds.FctInsertKeyValue != nil {
		return mds.FctInsertKeyValue(key, value)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) GetKeyValue(key string) (string, error) {
	if mds.FctGetKeyValue != nil {
		return mds.FctGetKeyValue(key)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) Lock(name string, owner string, duration time.Duration, renew bool) (bool, time.Time) {
	if mds.FctLock != nil {
		return mds.FctLock(name, owner, duration, renew)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) Unlock(name, owner string) {
	if mds.FctUnlock != nil {
		mds.FctUnlock(name, owner)
		return
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) FindLock(name string) (string, time.Time, error) {
	if mds.FctFindLock != nil {
		return mds.FctFindLock(name)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) Ping() bool {
	if mds.FctPing != nil {
		return mds.FctPing()
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) Close() {
	if mds.FctClose != nil {
		mds.FctClose()
		return
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) GetLayerBySHA(sha string, opts *DatastoreOptions) (string, string, bool, error) {
	panic("required mock function not implemented")
}

func (mds *MockDatastore) GetLayerByName(name string, opts *DatastoreOptions) (string, string, bool, error) {
	panic("required mock function not implemented")
}

func (mds *MockDatastore) AddImage(layer, lineage, digest, name string, opts *DatastoreOptions) error {
	panic("required mock function not implemented")
}

func (mds *MockDatastore) InsertLayerComponents(l, lineage string, c []*component.Component, r []string, opts *DatastoreOptions) error {
	if mds.FctInsertLayerComponents != nil {
		return mds.FctInsertLayerComponents(l, lineage, c, r, opts)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) GetLayerLanguageComponents(layer, lineage string, opts *DatastoreOptions) ([]*component.LayerToComponents, error) {
	if mds.FctGetLayerLanguageComponents != nil {
		return mds.FctGetLayerLanguageComponents(layer, lineage, opts)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) GetVulnerabilitiesForFeatureVersion(featureVersion FeatureVersion) ([]Vulnerability, error) {
	if mds.FctGetVulnerabilitiesForFeatureVersion != nil {
		return mds.FctGetVulnerabilitiesForFeatureVersion(featureVersion)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) LoadVulnerabilities(featureVersions []FeatureVersion) error {
	if mds.FctLoadVulnerabilities != nil {
		return mds.FctLoadVulnerabilities(featureVersions)
	}
	panic("required mock function not implemented")
}

func (mds *MockDatastore) FeatureExists(namespace, feature string) (bool, error) {
	if mds.FctFeatureExists != nil {
		return mds.FctFeatureExists(namespace, feature)
	}
	panic("required mock function not implemented")
}
