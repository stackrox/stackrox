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

package database

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/archop"
)

// Model defines the base type for each database entity.
type Model struct {
	// ID is only meant to be used by database implementations and should never be used for anything else.
	ID int `json:"id,omitempty" hash:"ignore"`
}

// Layer is an image layer.
type Layer struct {
	Model

	Name          string
	EngineVersion int
	Parent        *Layer
	Namespace     *Namespace
	Distroless    bool
	Features      []FeatureVersion
}

// Namespace is an image's OS.
type Namespace struct {
	Model

	Name          string
	VersionFormat string
}

// Feature is scanned package.
type Feature struct {
	Model

	Name       string
	Namespace  Namespace
	SourceType string
	Location   string
}

// FeatureVersion is the full result of a scanned package.
type FeatureVersion struct {
	Model

	Feature    Feature
	Version    string
	AffectedBy []Vulnerability
	// ExecutableToDependencies maps a feature provided executable to its dependencies.
	// Eg, If executable E is provided by this feature, and it imports a library B, we will have a map for E -> [B]
	ExecutableToDependencies StringToStringsMap
	// LibraryToDependencies maps a feature provided library to its dependencies.
	// Eg, If library A is provided by this feature, and it imports a library B, we will have a map for A -> [B]
	LibraryToDependencies StringToStringsMap

	// For internal purposes, only.
	Executables []*v1.Executable

	// For output purposes. Only make sense when the feature version is in the context of an image.
	AddedBy Layer
}

// Vulnerability defines a package vulnerability.
type Vulnerability struct {
	Model

	Name      string
	Namespace Namespace

	Description string
	Link        string
	Severity    Severity

	Metadata MetadataMap

	FixedIn []FeatureVersion

	// For output purposes. Only make sense when the vulnerability
	// is already about a specific Feature/FeatureVersion.
	FixedBy string `json:",omitempty"`

	SubCVEs []string
}

// MetadataMap represents vulnerability metadata.
type MetadataMap map[string]interface{}

// Scan writes the given SQL value into the caller.
func (mm *MetadataMap) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	// github.com/lib/pq decodes TEXT/VARCHAR fields into strings.
	val, ok := value.(string)
	if !ok {
		panic("got type other than []byte from database")
	}
	return json.Unmarshal([]byte(val), mm)
}

// Value converts the MetadataMap into a driver.Value.
func (mm *MetadataMap) Value() (driver.Value, error) {
	json, err := json.Marshal(*mm)
	return string(json), err
}

///////////////////////////////////////////////////
// BEGIN
// Influenced by ClairCore under Apache 2.0 License
// https://github.com/quay/claircore
///////////////////////////////////////////////////

// RHELv2Vulnerability represents a RHEL vulnerability
// from the OVAL v2 feeds as part of the Red Hat Scanner Certification program.
type RHELv2Vulnerability struct {
	Model

	Name         string               `json:"name"`
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	Issued       time.Time            `json:"issued"`
	Updated      time.Time            `json:"updated"`
	Link         string               `json:"link"`
	Severity     string               `json:"severity"`
	CVSSv3       string               `json:"cvssv3,omitempty"`
	CVSSv2       string               `json:"cvssv2,omitempty"`
	CPEs         []string             `json:"cpes" hash:"ignore"` // These are checked explcitly due to the removal of unused CPEs
	PackageInfos []*RHELv2PackageInfo `json:"package_info" hash:"set"`
	SubCVEs      []string             `json:"sub_cves,omitempty" hash:"set"`
}

// RHELv2PackageInfo defines all the data necessary for fully define a RHELv2 package.
type RHELv2PackageInfo struct {
	Packages       []*RHELv2Package `json:"package" hash:"set"`
	FixedInVersion string           `json:"fixed_in_version"`
	ArchOperation  archop.ArchOp    `json:"arch_op,omitempty"`
}

// RHELv2Package defines the basic information of a RHELv2 package.
type RHELv2Package struct {
	Model

	Name            string `json:"name"`
	Version         string `json:"version,omitempty"`
	Module          string `json:"module,omitempty"`
	Arch            string `json:"arch,omitempty"`
	ResolutionState string `json:"resolution_state"`

	// ExecutableToDependencies maps a feature provided executable to its dependencies.
	// Eg, If executable E is provided by this feature, and it imports a library B, we will have a map for E -> [B]
	ExecutableToDependencies StringToStringsMap `json:"executable_to_dependencies,omitempty"`
	// LibraryToDependencies maps a feature provided library to its dependencies.
	// Eg, If library A is provided by this feature, and it imports a library B, we will have a map for A -> [B]
	LibraryToDependencies StringToStringsMap `json:"library_to_dependencies,omitempty"`
	// Executables lists the executables determined from ExecutableToDependencies and
	// LibraryToDependencies. This is only populated when both ExecutableToDependencies and
	// LibraryToDependencies are empty.
	Executables []*v1.Executable `json:"executables,omitempty"`
}

func (p *RHELv2Package) String() string {
	return strings.Join([]string{p.Name, p.Version, p.Module, p.Arch}, ":")
}

// GetPackageVersion concatenates version and arch and returns package version
func (p *RHELv2Package) GetPackageVersion() string {
	version := p.Version
	if p.Arch != "" {
		version = version + "." + p.Arch
	}
	return version
}

// RHELv2Layer defines a RHEL image layer.
type RHELv2Layer struct {
	Model

	Hash       string
	ParentHash string
	Dist       string
	Pkgs       []*RHELv2Package
	CPEs       []string
}

// RHELv2Components defines the RHELv2 components found in a layer.
type RHELv2Components struct {
	Dist     string
	Packages []*RHELv2Package
	CPEs     []string
}

func (r *RHELv2Components) String() string {
	var buf bytes.Buffer
	buf.WriteString(r.Dist)
	buf.WriteString(" - ")
	buf.WriteString("[ ")
	for _, cpe := range r.CPEs {
		buf.WriteString(cpe)
		buf.WriteString(" ")
	}
	buf.WriteString("]")
	buf.WriteString("[ ")
	for _, pkg := range r.Packages {
		buf.WriteString(pkg.String())
		buf.WriteString(" ")
	}
	buf.WriteString("]")

	return buf.String()
}

// RHELv2PackageEnv contains a RHELv2Package plus
// data about the environment surrounding a particular package.
type RHELv2PackageEnv struct {
	Pkg       *RHELv2Package
	Namespace string
	AddedBy   string
	CPEs      []string
}

// RHELv2Record is used for querying RHELv2 vulnerabilities from the database.
type RHELv2Record struct {
	Pkg *RHELv2Package
	CPE string
}

// ContentManifest structure is based on file provided by OSBS
// The struct stores content metadata about the image
type ContentManifest struct {
	ContentSets []string         `json:"content_sets"`
	Metadata    ManifestMetadata `json:"metadata"`
}

// ManifestMetadata struct holds additional metadata about image
type ManifestMetadata struct {
	ImageLayerIndex int `json:"image_layer_index"`
}

///////////////////////////////////////////////////
// END
// Influenced by ClairCore under Apache 2.0 License
// https://github.com/quay/claircore
///////////////////////////////////////////////////
