// Package scos identifies CentOS Stream CoreOS node compatibility profiles.
package scos

import (
	"regexp"
	"strings"

	"github.com/stackrox/rox/pkg/features"
)

const (
	// Name is the OS image name reported by kubelet for CentOS Stream CoreOS.
	Name = "CentOS Stream CoreOS"
	// DistributionID distinguishes SCOS from RHEL and RHCOS in scanner reports.
	DistributionID = "centos-stream-coreos"
)

var kernelRelease = regexp.MustCompile(`\.el(9|10)(?:_[0-9]+)?\.(x86_64|aarch64|ppc64le|s390x)$`)

// Profile identifies the RHEL major release and RPM architecture used for matching.
type Profile struct {
	Major string
	Arch  string
}

// Detect identifies stock SCOS kernels. Older SCOS OS images contain an OKD
// build number, so the EL release tag in the kernel identifies the major version.
func Detect(osImage, kernel string) (Profile, bool) {
	if osImage != Name && !strings.HasPrefix(osImage, Name+" ") {
		return Profile{}, false
	}
	match := kernelRelease.FindStringSubmatch(kernel)
	if match == nil {
		return Profile{}, false
	}
	return Profile{Major: match[1], Arch: match[2]}, true
}

// Enabled reports whether Central can enable SCOS matching for node indexes.
func Enabled() bool {
	return features.SCOSNodeScanning.Enabled() && features.NodeIndexEnabled.Enabled() && features.ScannerV4.Enabled()
}
