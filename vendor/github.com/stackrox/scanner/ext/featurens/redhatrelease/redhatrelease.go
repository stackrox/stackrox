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

// Package redhatrelease implements a featurens.Detector for container image
// layers containing an redhat-release-like files.
//
// This detector is typically useful for detecting CentOS and Red-Hat like
// systems.
package redhatrelease

import (
	"regexp"
	"strings"

	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/ext/featurens"
	"github.com/stackrox/scanner/ext/versionfmt/rpm"
	"github.com/stackrox/scanner/pkg/analyzer"
)

var (
	amazonReleaseRegexp = regexp.MustCompile(`(?P<os>Amazon) (Linux release|Linux AMI release) (?P<version>[\d]+\.[\d]+|[\d]+)`)
	oracleReleaseRegexp = regexp.MustCompile(`(?P<os>Oracle) (Linux Server release) (?P<version>[\d]+)`)
	centosReleaseRegexp = regexp.MustCompile(`(?P<os>[^\s]*) (Linux release|release) (?P<version>[\d]+)`)
	redhatReleaseRegexp = regexp.MustCompile(`(?P<os>Red Hat Enterprise Linux) (Client release|Server release|Workstation release|release) (?P<version>[\d]+)`)

	// RequiredFilenames defines the names of the files required to identify the RHEL-based release.
	RequiredFilenames = []string{"etc/oracle-release", "etc/centos-release", "etc/redhat-release", "etc/system-release"}
)

type detector struct{}

func init() {
	featurens.RegisterDetector("redhat-release", &detector{})
}

func (d detector) Detect(files analyzer.Files, opts *featurens.DetectorOptions) *database.Namespace {
	for _, filePath := range d.RequiredFilenames() {
		f, hasFile := files.Get(filePath)
		if !hasFile {
			continue
		}

		var r []string

		// Attempt to match Amazon Linux.
		r = amazonReleaseRegexp.FindStringSubmatch(string(f.Contents))
		if len(r) == 4 {
			// Amazon Linux's namespace name should be amzn but the
			// /etc/system-release file uses Amazon.
			return &database.Namespace{
				Name:          "amzn" + ":" + r[3],
				VersionFormat: rpm.ParserName,
			}
		}

		// Attempt to match Oracle Linux.
		r = oracleReleaseRegexp.FindStringSubmatch(string(f.Contents))
		if len(r) == 4 {
			return &database.Namespace{
				Name:          strings.ToLower(r[1]) + ":" + r[3],
				VersionFormat: rpm.ParserName,
			}
		}

		// Attempt to match RHEL.
		r = redhatReleaseRegexp.FindStringSubmatch(string(f.Contents))
		if len(r) == 4 {
			namespace := &database.Namespace{
				VersionFormat: rpm.ParserName,
			}
			if opts.GetUncertifiedRHEL() {
				namespace.Name = "centos:" + r[3]
			} else {
				namespace.Name = "rhel:" + r[3]
			}
			return namespace
		}

		// Attempt to match CentOS.
		r = centosReleaseRegexp.FindStringSubmatch(string(f.Contents))
		if len(r) == 4 {
			return &database.Namespace{
				Name:          strings.ToLower(r[1]) + ":" + r[3],
				VersionFormat: rpm.ParserName,
			}
		}
	}

	return nil
}

func (d detector) RequiredFilenames() []string {
	return RequiredFilenames
}
