package cmd

import (
	"os"

	"github.com/stackrox/rox/pkg/buildinfo"
)

// fakeReportProviderEnabled switches roxagent to serve canned reports instead
// of real host-scan results, for pull-mode vsock scale/load testing.
//
// LOAD-TEST ONLY, interim mechanism — revisit before merging to master.
// Deliberately a raw, undocumented os.Getenv() read (not registered through
// pkg/env's settings framework, so it never appears in generated settings
// docs), additionally gated by buildinfo.ReleaseBuild so it structurally
// cannot activate in a real release build regardless of the env var.
var fakeReportProviderEnabled = !buildinfo.ReleaseBuild && os.Getenv("ROX_VM_VSOCK_LOADTEST_FAKE_REPORTS") == "true"
