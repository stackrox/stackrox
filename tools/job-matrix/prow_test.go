package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_clusterTypeForTest(t *testing.T) {
	tests := map[string]struct {
		workflow       string
		clusterProfile string
		ocpVersion     string
		want           string
	}{
		"rosa hcp": {
			workflow: "stackrox-automation-flavors-rosa-hcp-e2e",
			want:     "ROSA HCP (Red Hat OpenShift on AWS, hosted control plane)",
		},
		"rosa classic": {
			workflow: "stackrox-automation-flavors-rosa-e2e",
			want:     "ROSA (Red Hat OpenShift on AWS)",
		},
		"aks": {
			workflow: "stackrox-automation-flavors-aks-e2e",
			want:     "AKS (Azure Kubernetes Service)",
		},
		"aro": {
			workflow: "stackrox-automation-flavors-aro-e2e",
			want:     "ARO (Azure Red Hat OpenShift)",
		},
		"osd aws": {
			workflow: "stackrox-automation-flavors-osd-aws-e2e",
			want:     "OpenShift Dedicated on AWS",
		},
		"osd gcp": {
			workflow: "stackrox-automation-flavors-osd-gcp-e2e",
			want:     "OpenShift Dedicated on GCP",
		},
		"eks": {
			workflow: "stackrox-automation-flavors-eks-e2e",
			want:     "EKS (Amazon Elastic Kubernetes Service)",
		},
		"ibm cloud z": {
			workflow: "stackrox-automation-flavors-ibmcloudz-e2e",
			want:     "IBM Cloud on IBM Z",
		},
		"power vs": {
			workflow: "stackrox-automation-flavors-powervs-e2e",
			want:     "IBM Power Systems Virtual Server",
		},
		"self-managed ocp with known version": {
			workflow:   "stackrox-automation-flavors-ocp-4-e2e",
			ocpVersion: "4.22",
			want:       "OpenShift 4.22 (self-managed, AWS IPI)",
		},
		"self-managed ocp with unknown version": {
			workflow: "stackrox-automation-flavors-ocp-4-e2e",
			want:     "OpenShift (self-managed, AWS IPI)",
		},
		"gke": {
			workflow: "stackrox-stackrox-e2e-job",
			want:     "GKE (Google Kubernetes Engine)",
		},
		"firewatch interop": {
			workflow: "stackrox-firewatch-ipi-aws-cr",
			want:     "OpenShift (Red Hat CSPI interop pipeline, AWS IPI)",
		},
		"cluster_profile fallback when workflow doesn't match a known provider": {
			workflow:       "stackrox-install-roxie",
			clusterProfile: "aws-stackrox",
			ocpVersion:     "perf-scale",
			want:           "OpenShift perf-scale (self-managed, AWS IPI)",
		},
		"unrecognized workflow and cluster_profile yields unknown": {
			workflow: "some-future-workflow",
			want:     "Unknown",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := clusterTypeForTest(tt.workflow, tt.clusterProfile, tt.ocpVersion)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_ocpVersionForBranch(t *testing.T) {
	tests := map[string]struct {
		branch string
		want   string
	}{
		"no variant suffix":        {branch: "master", want: ""},
		"dash version":             {branch: "master__ocp-4-22", want: "4.22"},
		"dash version, two digits": {branch: "master__ocp-5-0", want: "5.0"},
		"dotted version in irregular variant name": {
			branch: "master__ocp-4.21-lp-interop-cr-acs-latest",
			want:   "4.21",
		},
		"non-numeric variant falls back to the raw label": {
			branch: "master__ocp-dev-preview",
			want:   "dev-preview",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, ocpVersionForBranch(tt.branch))
		})
	}
}

func Test_testFamily(t *testing.T) {
	// A representative sample of the real Prow test names found in
	// tmp/prow-jobs.html, hand-verified to share an underlying suite.
	tests := map[string]string{
		"qa-e2e-tests":                              "qa-e2e-tests",
		"gke-qa-e2e-tests":                          "qa-e2e-tests",
		"merge-gke-qa-e2e-tests":                    "qa-e2e-tests",
		"gke-latest-qa-e2e-tests":                   "qa-e2e-tests",
		"merge-gke-oldest-qa-e2e-tests":             "qa-e2e-tests",
		"aks-qa-e2e-tests":                          "qa-e2e-tests",
		"aro-qa-e2e-tests":                          "qa-e2e-tests",
		"eks-qa-e2e-tests":                          "qa-e2e-tests",
		"osd-aws-qa-e2e-tests":                      "qa-e2e-tests",
		"osd-gcp-qa-e2e-tests":                      "qa-e2e-tests",
		"rosa-qa-e2e-tests":                         "qa-e2e-tests",
		"rosa-hcp-qa-e2e-tests":                     "qa-e2e-tests",
		"rosa-fips-qa-e2e-tests":                    "qa-e2e-tests",
		"fips-qa-e2e-tests":                         "qa-e2e-tests",
		"merge-fips-qa-e2e-tests":                   "qa-e2e-tests",
		"ibmcloudz-4-14-qa-e2e-tests":               "qa-e2e-tests",
		"daily-ibmcloudz-latest-qa-e2e-tests":       "qa-e2e-tests",
		"daily-ibmcloudz-lts-qa-e2e-tests":          "qa-e2e-tests",
		"powervs-qa-e2e-tests":                      "qa-e2e-tests",
		"powervs-4-18-qa-corebpf-e2e-tests":         "qa-e2e-tests",
		"daily-powervs-stable-corebpf-qa-e2e-tests": "qa-e2e-tests",
		"gke-race-condition-qa-e2e-tests":           "qa-e2e-tests",
		"gke-external-pg-17-qa-e2e-tests":           "qa-e2e-tests",
		"ui-e2e-tests":                              "ui-e2e-tests",
		"gke-oldest-ui-e2e-tests":                   "ui-e2e-tests",
		"operator-e2e-tests":                        "operator-e2e-tests",
		"merge-gke-latest-operator-e2e-tests":       "operator-e2e-tests",
		"nongroovy-e2e-tests":                       "nongroovy-e2e-tests",
		"merge-gke-oldest-nongroovy-e2e-tests":      "nongroovy-e2e-tests",
		"gke-nongroovy-compatibility-tests":         "nongroovy-compatibility-tests",
		"gke-version-compatibility-tests":           "version-compatibility-tests",
		"vm-scanning-e2e-tests":                     "vm-scanning-e2e-tests",
		"ocp-vm-scanning-e2e-tests":                 "vm-scanning-e2e-tests",
		"scanner-v4-install-tests":                  "scanner-v4-install-tests",
		"merge-gke-scanner-v4-install-tests":        "scanner-v4-install-tests",
		"compliance-e2e-tests":                      "compliance-e2e-tests",
		"merge-compliance-e2e-tests":                "compliance-e2e-tests",
		"gke-upgrade-tests":                         "upgrade-tests",
		"merge-gke-upgrade-tests":                   "upgrade-tests",
		"gke-scale-tests":                           "scale-tests",
		"24nodes-scale-test":                        "scale-test",
		"24nodes-scale-scannerv4-test":              "scale-scannerv4-test",
		"cr--acs--tests-aws":                        "acs-interop-tests",
		"acs-tests-aws":                             "acs-interop-tests",
		"acs--tests-aws-fips":                       "acs-interop-tests",
		"acs-tests-aws-fips":                        "acs-interop-tests",
	}
	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, want, testFamily(name))
		})
	}
}

func Test_isNightlyOnly(t *testing.T) {
	tests := map[string]struct {
		branches []string
		want     bool
	}{
		"no branches":                  {branches: nil, want: false},
		"all nightlies variants":       {branches: []string{"nightlies", "nightlies__ocp-4-12"}, want: true},
		"plain master":                 {branches: []string{"master"}, want: false},
		"mixed nightlies and release":  {branches: []string{"nightlies", "release-4.10"}, want: false},
		"release branch, no nightlies": {branches: []string{"release-4.10", "release-4.11"}, want: false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, isNightlyOnly(tt.branches))
		})
	}
}

func Test_triggersForProwTest(t *testing.T) {
	tests := map[string]struct {
		branches   []string
		cron       string
		postsubmit bool
		want       []Trigger
	}{
		"plain presubmit": {
			branches: []string{"master"},
			want:     []Trigger{TriggerPreMerge},
		},
		"postsubmit on a normal branch": {
			branches:   []string{"release-4.10"},
			postsubmit: true,
			want:       []Trigger{TriggerPostMerge},
		},
		"cron on a normal branch is periodic": {
			branches: []string{"master"},
			cron:     "0 3 * * *",
			want:     []Trigger{TriggerPeriodic},
		},
		"postsubmit under nightlies is nightly, not post-merge": {
			branches:   []string{"nightlies"},
			postsubmit: true,
			want:       []Trigger{TriggerNightly},
		},
		"cron under nightlies is still nightly, not periodic": {
			branches: []string{"nightlies__perf-scale"},
			cron:     "0 6 * * *",
			want:     []Trigger{TriggerNightly},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, triggersForProwTest(tt.branches, tt.cron, tt.postsubmit))
		})
	}
}

func Test_htmlFileProwSource_Tests(t *testing.T) {
	const fixture = `<!doctype html>
<html><body>
<h4 id="config-summary">Config summary</h4>
<table>
<thead>
<tr><th><div></div></th><th><div>master</div></th><th><div>nightlies</div></th><th><div>release-4.10</div></th></tr>
</thead>
<tbody>
<tr><td class="header-cell">tests</td><td colspan="3"></td></tr>
<tr class="None"><td class="right-aligned"><div>qa-e2e-tests</div></td><td class="None"><div><div class="collapse"><pre>as: qa-e2e-tests
steps:
  workflow: stackrox-automation-flavors-ocp-4-e2e
  env:
    TEST_SUITE: ocp-qa-e2e-tests
</pre></div></div></td><td class="None"><div></div></td><td class="None"><div></div></td></tr>
<tr class="None"><td class="right-aligned"><div>opt-in-qa-e2e-tests</div></td><td class="None"><div><pre>as: opt-in-qa-e2e-tests
always_run: false
optional: true
steps:
  workflow: stackrox-automation-flavors-ocp-4-e2e
</pre></div></td><td class="None"><div></div></td><td class="None"><div></div></td></tr>
<tr class="None"><td class="right-aligned"><div>gke-qa-e2e-tests</div></td><td class="None"><div></div></td><td class="None"><div></div></td><td class="None"><div><pre>as: gke-qa-e2e-tests
postsubmit: true
steps:
  workflow: stackrox-stackrox-e2e-job
</pre></div></td></tr>
<tr class="None"><td class="right-aligned"><div>nightly-postsubmit-test</div></td><td class="None"><div></div></td><td class="None"><div><pre>as: nightly-postsubmit-test
postsubmit: true
steps:
  workflow: stackrox-stackrox-e2e-job
</pre></div></td><td class="None"><div></div></td></tr>
<tr class="None"><td class="right-aligned"><div>24nodes-scale-scannerv4-test</div></td><td class="None"><div></div></td><td class="None"><div><pre>as: 24nodes-scale-scannerv4-test
cron: 0 6 * * *
steps:
  cluster_profile: aws-stackrox
  workflow: stackrox-install-roxie
</pre></div></td><td class="None"><div></div></td></tr>
<tr class="None"><td class="right-aligned"><div>cr--acs--tests-aws</div></td><td class="None"><div><pre>as: cr--acs--tests-aws
cron: 0 3,15 * * *
steps:
  cluster_profile: aws-cspi-qe
  workflow: stackrox-firewatch-ipi-aws-cr
</pre></div></td><td class="None"><div></div></td><td class="None"><div></div></td></tr>
</tbody>
</table>
</body></html>
`
	path := filepath.Join(t.TempDir(), "prow-jobs.html")
	require.NoError(t, os.WriteFile(path, []byte(fixture), 0o600))

	tests, err := NewHTMLFileProwSource(path).Tests()
	require.NoError(t, err)
	require.Len(t, tests, 6)

	byName := map[string]ProwTest{}
	for _, test := range tests {
		byName[test.Name] = test
	}

	qaE2E := byName["qa-e2e-tests"]
	assert.Equal(t, []string{"master"}, qaE2E.Branches)
	assert.Equal(t, "OpenShift (self-managed, AWS IPI)", qaE2E.ClusterType)
	assert.Equal(t, []Trigger{TriggerPreMerge}, qaE2E.Triggers)
	assert.True(t, qaE2E.RunsAutomatically)
	assert.False(t, qaE2E.Optional)
	assert.Equal(t, "qa-e2e-tests", qaE2E.TestFamily)

	optIn := byName["opt-in-qa-e2e-tests"]
	assert.Equal(t, []Trigger{TriggerPreMerge}, optIn.Triggers)
	assert.False(t, optIn.RunsAutomatically)
	assert.True(t, optIn.Optional)

	gkeQAE2E := byName["gke-qa-e2e-tests"]
	assert.Equal(t, []string{"release-4.10"}, gkeQAE2E.Branches)
	assert.Equal(t, "GKE (Google Kubernetes Engine)", gkeQAE2E.ClusterType)
	assert.Equal(t, []Trigger{TriggerPostMerge}, gkeQAE2E.Triggers)
	assert.Equal(t, "qa-e2e-tests", gkeQAE2E.TestFamily)

	nightlyPostsubmit := byName["nightly-postsubmit-test"]
	assert.Equal(t, []string{"nightlies"}, nightlyPostsubmit.Branches)
	assert.Equal(t, []Trigger{TriggerNightly}, nightlyPostsubmit.Triggers)

	nightlyPeriodic := byName["24nodes-scale-scannerv4-test"]
	assert.Equal(t, []string{"nightlies"}, nightlyPeriodic.Branches)
	// Even though this one has a cron:, it's under "nightlies" and so is
	// still categorized as nightly rather than periodic.
	assert.Equal(t, []Trigger{TriggerNightly}, nightlyPeriodic.Triggers)

	acsInterop := byName["cr--acs--tests-aws"]
	assert.Equal(t, []string{"master"}, acsInterop.Branches)
	assert.Equal(t, "OpenShift (Red Hat CSPI interop pipeline, AWS IPI)", acsInterop.ClusterType)
	assert.Equal(t, []Trigger{TriggerPeriodic}, acsInterop.Triggers)
	assert.Equal(t, "acs-interop-tests", acsInterop.TestFamily)
}
