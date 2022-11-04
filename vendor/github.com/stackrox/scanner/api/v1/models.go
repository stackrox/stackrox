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

package v1

import (
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/scanner/api/v1/common"
	"github.com/stackrox/scanner/api/v1/convert"
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/ext/featurefmt"
	"github.com/stackrox/scanner/ext/versionfmt"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/component"
	"github.com/stackrox/scanner/pkg/env"
	"github.com/stackrox/scanner/pkg/rhel"
	namespaces "github.com/stackrox/scanner/pkg/wellknownnamespaces"
)

// Linux and kernel packages that are not applicable to images
var kernelPrefixes = []string{
	"linux",
	"kernel",
}

// Error is a scanning error.
type Error struct {
	Message string `json:"Message,omitempty"`
}

// Layer is an image layer.
type Layer struct {
	Name             string            `json:"Name,omitempty"`
	NamespaceName    string            `json:"NamespaceName,omitempty"`
	Path             string            `json:"Path,omitempty"`
	Headers          map[string]string `json:"Headers,omitempty"`
	ParentName       string            `json:"ParentName,omitempty"`
	Format           string            `json:"Format,omitempty"`
	IndexedByVersion int               `json:"IndexedByVersion,omitempty"`
	Features         []Feature         `json:"Features,omitempty"`
}

// vulnerabilityFromDatabaseModel converts the given database.Vulnerability into a Vulnerability.
func vulnerabilityFromDatabaseModel(dbVuln database.Vulnerability) Vulnerability {
	vuln := Vulnerability{
		Name:          dbVuln.Name,
		NamespaceName: dbVuln.Namespace.Name,
		Description:   dbVuln.Description,
		Link:          dbVuln.Link,
		Severity:      string(convert.DatabaseSeverityToSeverity(dbVuln.Severity)),
		Metadata:      dbVuln.Metadata,
	}
	if dbVuln.FixedBy != versionfmt.MaxVersion {
		vuln.FixedBy = dbVuln.FixedBy
	}
	return vuln
}

func featureFromDatabaseModel(dbFeatureVersion database.FeatureVersion, uncertified bool, depMap map[string]common.FeatureKeySet) *Feature {
	version := dbFeatureVersion.Version
	if version == versionfmt.MaxVersion {
		version = "None"
	}

	addedBy := dbFeatureVersion.AddedBy.Name
	if uncertified {
		addedBy = rhel.GetOriginalLayerName(addedBy)
	}

	featureKey := featurefmt.PackageKey{Name: dbFeatureVersion.Feature.Name, Version: dbFeatureVersion.Version}
	executables := common.CreateExecutablesFromDependencies(featureKey, dbFeatureVersion.ExecutableToDependencies, depMap)
	return &Feature{
		Name:          dbFeatureVersion.Feature.Name,
		NamespaceName: dbFeatureVersion.Feature.Namespace.Name,
		VersionFormat: stringutils.OrDefault(dbFeatureVersion.Feature.SourceType, dbFeatureVersion.Feature.Namespace.VersionFormat),
		Version:       version,
		AddedBy:       addedBy,
		Location:      dbFeatureVersion.Feature.Location,
		Executables:   executables,
	}
}

func hasKernelPrefix(name string) bool {
	for _, prefix := range kernelPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// LayerFromDatabaseModel returns the scan data for the given layer based on the data in the given datastore.
func LayerFromDatabaseModel(db database.Datastore, dbLayer database.Layer, lineage string, depMap map[string]common.FeatureKeySet, opts *database.DatastoreOptions) (Layer, []Note, error) {
	withFeatures := opts.GetWithFeatures()
	withVulnerabilities := opts.GetWithVulnerabilities()
	uncertifiedRHEL := opts.GetUncertifiedRHEL()
	layer := Layer{
		Name:             dbLayer.Name,
		IndexedByVersion: dbLayer.EngineVersion,
	}

	if dbLayer.Parent != nil {
		layer.ParentName = dbLayer.Parent.Name
	}

	if dbLayer.Namespace != nil {
		layer.NamespaceName = dbLayer.Namespace.Name
	}

	notes := getNotes(layer.NamespaceName, uncertifiedRHEL)

	if (withFeatures || withVulnerabilities) && (dbLayer.Features != nil || namespaces.IsRHELNamespace(layer.NamespaceName)) {
		for _, dbFeatureVersion := range dbLayer.Features {
			feature := featureFromDatabaseModel(dbFeatureVersion, opts.GetUncertifiedRHEL(), depMap)

			if hasKernelPrefix(feature.Name) {
				continue
			}

			updateFeatureWithVulns(feature, dbFeatureVersion.AffectedBy, dbFeatureVersion.Feature.Namespace.VersionFormat)
			layer.Features = append(layer.Features, *feature)
		}
		if !uncertifiedRHEL && namespaces.IsRHELNamespace(layer.NamespaceName) {
			certified, err := addRHELv2Vulns(db, &layer)
			if err != nil {
				return layer, notes, err
			}
			if !certified {
				// Client expected certified results, but they are unavailable.
				notes = append(notes, CertifiedRHELScanUnavailable)
			}
		}
		if env.LanguageVulns.Enabled() {
			addLanguageVulns(db, &layer, lineage, uncertifiedRHEL)
		}
	}

	return layer, notes, nil
}

func updateFeatureWithVulns(feature *Feature, dbVulns []database.Vulnerability, versionFormat string) {
	allVulnsFixedBy := feature.FixedBy
	for _, dbVuln := range dbVulns {
		vuln := vulnerabilityFromDatabaseModel(dbVuln)
		feature.Vulnerabilities = append(feature.Vulnerabilities, vuln)

		// If at least one vulnerability is not fixable, then we mark it the component as not fixable.
		if vuln.FixedBy == "" {
			continue
		}

		higherVersion, err := versionfmt.GetHigherVersion(versionFormat, vuln.FixedBy, allVulnsFixedBy)
		if err != nil {
			log.Errorf("comparing feature versions for %s: %v", feature.Name, err)
			continue
		}
		allVulnsFixedBy = higherVersion
	}
	feature.FixedBy = allVulnsFixedBy
}

// ComponentsFromDatabaseModel returns the package features and language components for the given layer based on the data in the given datastore.
//
// Two language components may potentially produce the same feature. Similarly, the feature may already be seen as an OS-package feature.
// However, these are not deduplicated here. This is left for the vulnerability matcher to determine upon converting the language components to feature versions.
func ComponentsFromDatabaseModel(db database.Datastore, dbLayer *database.Layer, lineage string, uncertifiedRHEL bool) (*ComponentsEnvelope, error) {
	var namespaceName string
	if dbLayer.Namespace != nil {
		namespaceName = dbLayer.Namespace.Name
	}

	var features []Feature
	var rhelv2PkgEnvs map[int]*database.RHELv2PackageEnv
	var components []*component.Component
	notes := getNotes(namespaceName, uncertifiedRHEL)

	if dbLayer.Features != nil {
		depMap := common.GetDepMap(dbLayer.Features)
		for _, dbFeatureVersion := range dbLayer.Features {
			feature := featureFromDatabaseModel(dbFeatureVersion, uncertifiedRHEL, depMap)

			if hasKernelPrefix(feature.Name) {
				continue
			}

			features = append(features, *feature)
		}
	}

	if !uncertifiedRHEL && namespaces.IsRHELNamespace(namespaceName) {
		var certified bool
		var err error
		rhelv2PkgEnvs, certified, err = getRHELv2PkgEnvs(db, dbLayer.Name)
		if err != nil {
			return nil, err
		}
		if !certified {
			// Client expected certified results, but they are unavailable.
			notes = append(notes, CertifiedRHELScanUnavailable)
		}
	}

	if env.LanguageVulns.Enabled() {
		components = getLanguageComponents(db, dbLayer.Name, lineage, uncertifiedRHEL)
	}

	return &ComponentsEnvelope{
		Namespace: namespaceName,

		Features:           features,
		RHELv2PkgEnvs:      rhelv2PkgEnvs,
		LanguageComponents: components,

		Notes: notes,
	}, nil
}

func getNotes(namespaceName string, uncertifiedRHEL bool) []Note {
	var notes []Note
	if namespaceName != "" {
		if namespaces.KnownStaleNamespaces.Contains(namespaceName) {
			notes = append(notes, OSCVEsStale)
		} else if !namespaces.KnownSupportedNamespaces.Contains(namespaceName) {
			notes = append(notes, OSCVEsUnavailable)
		}
	} else {
		notes = append(notes, OSCVEsUnavailable)
	}
	if !env.LanguageVulns.Enabled() {
		notes = append(notes, LanguageCVEsUnavailable)
	}
	if uncertifiedRHEL {
		// Uncertified results were requested.
		notes = append(notes, CertifiedRHELScanUnavailable)
	}

	return notes
}

// GetVulnerabilitiesForComponents retrieves the vulnerabilities for the given components.
func GetVulnerabilitiesForComponents(db database.Datastore, components *v1.Components, uncertifiedRHEL bool) (*Layer, error) {
	layer := &Layer{
		NamespaceName: components.GetNamespace(),
	}

	osFeatures, err := getOSFeatures(db, components.GetOsComponents())
	if err != nil {
		return nil, errors.Wrap(err, "getting OS features")
	}
	layer.Features = append(layer.Features, osFeatures...)

	if !uncertifiedRHEL {
		rhelv2Features, err := getFullFeaturesForRHELv2Packages(db, components.GetRhelComponents())
		if err != nil {
			return nil, errors.Wrap(err, "getting RHELv2 features")
		}
		layer.Features = append(layer.Features, rhelv2Features...)
	}

	languageFeatures, err := getLanguageFeatures(layer.Features, components.GetLanguageComponents(), uncertifiedRHEL)
	if err != nil {
		return nil, errors.Wrap(err, "getting language features")
	}
	layer.Features = append(layer.Features, languageFeatures...)

	return layer, nil
}

func getOSFeatures(db database.Datastore, components []*v1.OSComponent) ([]Feature, error) {
	featureVersions := osComponentsToFeatureVersions(components)

	err := db.LoadVulnerabilities(featureVersions)
	if err != nil {
		return nil, errors.Wrap(err, "loading OS vulnerabilities from database")
	}

	features := make([]Feature, 0, len(featureVersions))
	for _, fv := range featureVersions {
		feature := Feature{
			Name:          fv.Feature.Name,
			NamespaceName: fv.Feature.Namespace.Name,
			VersionFormat: versionfmt.GetVersionFormatForNamespace(fv.Feature.Namespace.Name),
			Version:       fv.Version,
			AddedBy:       fv.AddedBy.Name,
			Executables:   fv.Executables,
		}
		updateFeatureWithVulns(&feature, fv.AffectedBy, feature.VersionFormat)

		features = append(features, feature)
	}

	return features, nil
}

func osComponentsToFeatureVersions(components []*v1.OSComponent) []database.FeatureVersion {
	featureVersions := make([]database.FeatureVersion, 0, len(components))
	for _, c := range components {
		featureVersions = append(featureVersions, database.FeatureVersion{
			Feature: database.Feature{
				Name: c.GetName(),
				Namespace: database.Namespace{
					Name:          c.GetNamespace(),
					VersionFormat: versionfmt.GetVersionFormatForNamespace(c.GetNamespace()),
				},
			},
			Version:     c.GetVersion(),
			Executables: c.GetExecutables(),
			AddedBy:     database.Layer{Name: c.GetAddedBy()},
		})
	}

	return featureVersions
}

// Namespace is the image's base OS.
type Namespace struct {
	Name          string `json:"Name,omitempty"`
	VersionFormat string `json:"VersionFormat,omitempty"`
}

// Vulnerability defines a vulnerability.
type Vulnerability struct {
	Name          string                 `json:"Name,omitempty"`
	NamespaceName string                 `json:"NamespaceName,omitempty"`
	Description   string                 `json:"Description,omitempty"`
	Link          string                 `json:"Link,omitempty"`
	Severity      string                 `json:"Severity,omitempty"`
	Metadata      map[string]interface{} `json:"Metadata,omitempty"`
	FixedBy       string                 `json:"FixedBy,omitempty"`
}

// Feature is a scanned package in an image.
type Feature struct {
	Name            string           `json:"Name,omitempty"`
	NamespaceName   string           `json:"NamespaceName,omitempty"`
	VersionFormat   string           `json:"VersionFormat,omitempty"`
	Version         string           `json:"Version,omitempty"`
	Vulnerabilities []Vulnerability  `json:"Vulnerabilities,omitempty"`
	AddedBy         string           `json:"AddedBy,omitempty"`
	Location        string           `json:"Location,omitempty"`
	FixedBy         string           `json:"FixedBy,omitempty"`
	Executables     []*v1.Executable `json:"Executables,omitempty"`
}

// DatabaseModel returns a database.FeatureVersion based on the caller Feature.
func (f Feature) DatabaseModel() (fv database.FeatureVersion, err error) {
	var version string
	if f.Version == "None" {
		version = versionfmt.MaxVersion
	} else {
		err = versionfmt.Valid(f.VersionFormat, f.Version)
		if err != nil {
			return
		}
		version = f.Version
	}

	fv = database.FeatureVersion{
		Feature: database.Feature{
			Name: f.Name,
			Namespace: database.Namespace{
				Name:          f.NamespaceName,
				VersionFormat: f.VersionFormat,
			},
		},
		Version: version,
	}

	return
}

// LayerEnvelope envelopes complete scan data to return to the client.
// easyjson:json
type LayerEnvelope struct {
	ScannerVersion string `json:"ScannerVersion,omitempty"`
	Layer          *Layer `json:"Layer,omitempty"`
	Notes          []Note `json:"Notes,omitempty"`
	Error          *Error `json:"Error,omitempty"`
}

// Note defines scanning notes.
//
//go:generate stringer -type=Note
type Note int

const (
	// OSCVEsUnavailable labels scans of images with unknown namespaces or obsolete namespaces.
	OSCVEsUnavailable Note = iota
	// OSCVEsStale labels scans of images with namespaces whose CVEs are known to be stale.
	OSCVEsStale
	// LanguageCVEsUnavailable labels scans of images with without language CVEs.
	// This is typically only populated when language CVEs are not enabled.
	LanguageCVEsUnavailable
	// CertifiedRHELScanUnavailable labels scans of RHEL-based images out-of-scope
	// of the Red Hat Certification program.
	// These images were made before June 2020, and they are missing content manifest JSON files.
	CertifiedRHELScanUnavailable

	// SentinelNote is a fake note which should ALWAYS be last to ensure the proto is up-to-date.
	SentinelNote
)

// VulnerabilityEnvelope envelopes complete vulnerability data to return to the client.
type VulnerabilityEnvelope struct {
	Vulnerability   *Vulnerability   `json:"Vulnerability,omitempty"`
	Vulnerabilities *[]Vulnerability `json:"Vulnerabilities,omitempty"`
	NextPage        string           `json:"NextPage,omitempty"`
	Error           *Error           `json:"Error,omitempty"`
}

// FeatureEnvelope envelopes complete feature data to return to the client.
type FeatureEnvelope struct {
	Feature  *Feature   `json:"Feature,omitempty"`
	Features *[]Feature `json:"Features,omitempty"`
	Error    *Error     `json:"Error,omitempty"`
}

// ComponentsEnvelope envelopes component data (OS-packages and language-level-packages).
type ComponentsEnvelope struct {
	Namespace string

	Features []Feature
	// RHELv2PkgEnvs maps the package ID to the related package environment.
	RHELv2PkgEnvs      map[int]*database.RHELv2PackageEnv
	LanguageComponents []*component.Component

	Notes []Note
}
