// Presence of this file is a workaround for the following error
// when running unit tests: 
// Run GOTAGS=release make go-unit-tests
// mkdir -p bin/{darwin_amd64,darwin_arm64,linux_amd64,linux_arm64,linux_ppc64le,linux_s390x,windows_amd64}
// + test-prep
// package github.com/stackrox/rox/operator/tests/chart: build constraints exclude all Go files in /__w/stackrox/stackrox/operator/tests/chart
package chart
