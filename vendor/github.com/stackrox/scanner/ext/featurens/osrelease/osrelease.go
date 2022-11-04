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

// Package osrelease implements a featurens.Detector for container image
// layers containing an os-release file.
//
// This detector is typically useful for detecting Debian or Ubuntu.
package osrelease

import (
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/ext/featurens"
	"github.com/stackrox/scanner/ext/versionfmt/dpkg"
	"github.com/stackrox/scanner/ext/versionfmt/rpm"
	"github.com/stackrox/scanner/pkg/analyzer"
	"github.com/stackrox/scanner/pkg/osrelease"
)

var (
	// blocklistFilenames are files that should exclude this detector.
	blocklistFilenames = []string{
		"etc/alpine-release",
		"etc/centos-release",
		"etc/fedora-release",
		"etc/oracle-release",
		"etc/redhat-release",
		"usr/lib/centos-release",
	}

	// RequiredFilenames defines the names of the files required to identify the release.
	RequiredFilenames = []string{"etc/os-release", "usr/lib/os-release"}
)

type detector struct{}

func init() {
	featurens.RegisterDetector("os-release", &detector{})
}

func (d detector) Detect(files analyzer.Files, _ *featurens.DetectorOptions) *database.Namespace {
	var OS, version string

	for _, filePath := range blocklistFilenames {
		if _, hasFile := files.Get(filePath); hasFile {
			return nil
		}
	}

	for _, filePath := range d.RequiredFilenames() {
		f, hasFile := files.Get(filePath)
		if !hasFile {
			continue
		}

		OS, version = osrelease.GetOSAndVersionFromOSRelease(f.Contents)
	}

	// Determine the VersionFormat.
	// This intentionally does not support alpine,
	// as this detector does not handle alpine correctly.
	var versionFormat string
	switch OS {
	case "debian", "ubuntu":
		versionFormat = dpkg.ParserName
	case "centos", "rhel", "amzn", "oracle":
		versionFormat = rpm.ParserName
	default:
		return nil
	}

	if OS != "" && version != "" {
		return &database.Namespace{
			Name:          OS + ":" + version,
			VersionFormat: versionFormat,
		}
	}
	return nil
}

func (d detector) RequiredFilenames() []string {
	return RequiredFilenames
}
