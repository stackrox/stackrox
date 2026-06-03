package clairclient

import (
	"encoding/json"
	"time"
)

// Manifest represents the request body for POST /indexer/api/v1/index_report.
type Manifest struct {
	Hash   string  `json:"hash"`
	Layers []Layer `json:"layers"`
}

// Layer represents an individual container layer.
type Layer struct {
	Hash    string              `json:"hash"`
	URI     string              `json:"uri"`
	Headers map[string][]string `json:"headers,omitempty"`
}

// IndexReport represents the response from indexer endpoints.
type IndexReport struct {
	ManifestHash  string                   `json:"manifest_hash"`
	State         string                   `json:"state"`
	Success       bool                     `json:"success"`
	Err           string                   `json:"err"`
	Packages      map[string]Package       `json:"packages"`
	Distributions map[string]Distribution  `json:"distributions"`
	Repositories  map[string]Repository    `json:"repository"`
	Environments  map[string][]Environment `json:"environments"`
}

// VulnerabilityReport represents the response from matcher endpoint.
type VulnerabilityReport struct {
	ManifestHash           string                       `json:"manifest_hash"`
	State                  string                       `json:"state"`
	Success                bool                         `json:"success"`
	Err                    string                       `json:"err"`
	Packages               map[string]Package           `json:"packages"`
	Distributions          map[string]Distribution      `json:"distributions"`
	Repositories           map[string]Repository        `json:"repository"`
	Environments           map[string][]Environment     `json:"environments"`
	Vulnerabilities        map[string]Vulnerability     `json:"vulnerabilities"`
	PackageVulnerabilities map[string][]string          `json:"package_vulnerabilities"`
	Enrichments            map[string][]json.RawMessage `json:"enrichments"`
}

// Package represents a package found in a container image.
type Package struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Version           string            `json:"version"`
	Kind              string            `json:"kind"`
	Arch              string            `json:"arch"`
	Source            *Package          `json:"source,omitzero"`
	Module            string            `json:"module,omitzero"`
	CPE               string            `json:"cpe,omitzero"`
	PackageDB         string            `json:"package_db,omitzero"`
	RepositoryHint    string            `json:"repository_hint,omitzero"`
	NormalizedVersion NormalizedVersion `json:"normalized_version,omitzero"`
}

// NormalizedVersion represents a parsed and normalized package version.
type NormalizedVersion struct {
	Kind string `json:"kind"`
	V    []int  `json:"V"`
}

// Distribution represents a Linux distribution.
type Distribution struct {
	ID              string `json:"id"`
	DID             string `json:"did"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	VersionCodeName string `json:"version_code_name"`
	VersionID       string `json:"version_id"`
	Arch            string `json:"arch"`
	CPE             string `json:"cpe"`
	PrettyName      string `json:"pretty_name"`
}

// Repository represents a package repository.
type Repository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
	URI  string `json:"uri"`
	CPE  string `json:"cpe"`
}

// Environment represents the environment where a package was found.
type Environment struct {
	PackageDB      string   `json:"package_db"`
	IntroducedIn   string   `json:"introduced_in"`
	DistributionID string   `json:"distribution_id,omitzero"`
	RepositoryIDs  []string `json:"repository_ids,omitzero"`
}

// Vulnerability represents a security vulnerability.
type Vulnerability struct {
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Description        string        `json:"description"`
	Issued             time.Time     `json:"issued"`
	Links              string        `json:"links"`
	Severity           string        `json:"severity"`
	NormalizedSeverity string        `json:"normalized_severity"`
	FixedInVersion     string        `json:"fixed_in_version"`
	Package            *Package      `json:"package,omitzero"`
	Distribution       *Distribution `json:"distribution,omitzero"`
	Repository         *Repository   `json:"repository,omitzero"`
	Updater            string        `json:"updater"`
}

// IndexState represents the state of an index operation.
type IndexState struct {
	State string `json:"state"`
}

// UpdateOperation represents a vulnerability database update operation.
type UpdateOperation struct {
	Ref     string    `json:"ref"`
	Updater string    `json:"updater"`
	Date    time.Time `json:"date"`
}
