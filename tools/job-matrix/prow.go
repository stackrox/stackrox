package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProwTest describes one named ci-operator test ("as:") and the set of
// branch/variant columns that run an identical copy of it.
type ProwTest struct {
	Name        string
	Branches    []string
	ClusterType string
	Triggers    []Trigger
	// RunsAutomatically is false when the presubmit is opt-in only
	// (ci-operator's always_run: false): it won't run on a PR unless
	// someone comments "/test <name>". Only meaningful when Triggers
	// contains TriggerPreMerge.
	RunsAutomatically bool
	// Optional is true when ci-operator's optional: true is set: the job
	// can run and fail without blocking merge.
	Optional    bool
	Description string
	TestFamily  string
}

// ProwSource provides the set of Prow ci-operator tests defined for
// stackrox/stackrox. It's an interface so the HTML-file-based implementation
// below can later be swapped for one that scrapes openshift/release
// directly, without changing anything that consumes ProwTest.
type ProwSource interface {
	Tests() ([]ProwTest, error)
}

// yamlConfigProwSource reads Prow ci-operator test definitions directly out
// of a repo's ci-operator/config/<org>/<repo> YAML files -- the same files
// openshift/release's config.py reads to render its HTML summary -- rather
// than running that script and parsing its output.
type yamlConfigProwSource struct {
	dir string
}

// NewYAMLConfigProwSource returns a ProwSource that reads ci-operator test
// definitions from the ci-operator config YAML files in dir.
func NewYAMLConfigProwSource(dir string) ProwSource {
	return yamlConfigProwSource{dir: dir}
}

// prowConfigFile is the top-level shape of a ci-operator config YAML file
// under ci-operator/config/<org>/<repo>/.
type prowConfigFile struct {
	Tests []prowTestConfig `yaml:"tests"`
}

// prowTestConfig mirrors the subset of the ci-operator test schema relevant
// to deriving a ProwTest.
type prowTestConfig struct {
	As string `yaml:"as"`
	// AlwaysRun defaults to true in ci-operator when omitted; it's a
	// pointer here so "not set" (true by default) is distinguishable from
	// an explicit "always_run: false" (opt-in via a /test comment).
	AlwaysRun  *bool  `yaml:"always_run"`
	Optional   bool   `yaml:"optional"`
	Postsubmit bool   `yaml:"postsubmit"`
	Cron       string `yaml:"cron"`
	Steps      struct {
		ClusterProfile string            `yaml:"cluster_profile"`
		Workflow       string            `yaml:"workflow"`
		Env            map[string]string `yaml:"env"`
		Test           []struct {
			Ref string `yaml:"ref"`
		} `yaml:"test"`
	} `yaml:"steps"`
}

func (s yamlConfigProwSource) Tests() ([]ProwTest, error) {
	var paths []string
	for _, pattern := range []string{"*.yml", "*.yaml"} {
		matches, err := filepath.Glob(filepath.Join(s.dir, pattern))
		if err != nil {
			return nil, fmt.Errorf("globbing %s in %s: %w", pattern, s.dir, err)
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)

	basenames := make([]string, len(paths))
	for i, p := range paths {
		basenames[i] = filepath.Base(p)
	}
	prefix := commonPrefix(basenames)

	type testVariant struct {
		name string
		key  string
	}
	var order []testVariant
	branchesFor := map[testVariant][]string{}
	configFor := map[testVariant]prowTestConfig{}

	for _, p := range paths {
		base := filepath.Base(p)
		shortName := strings.TrimSuffix(strings.TrimPrefix(base, prefix), filepath.Ext(base))

		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", p, err)
		}
		var cfgFile prowConfigFile
		if err := yaml.Unmarshal(data, &cfgFile); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", p, err)
		}

		for _, cfg := range cfgFile.Tests {
			if cfg.As == "" {
				continue
			}
			tv := testVariant{name: cfg.As, key: prowTestConfigKey(cfg)}
			if _, seen := branchesFor[tv]; !seen {
				order = append(order, tv)
				configFor[tv] = cfg
			}
			branchesFor[tv] = append(branchesFor[tv], shortName)
		}
	}

	tests := make([]ProwTest, 0, len(order))
	for _, tv := range order {
		cfg := configFor[tv]
		branches := branchesFor[tv]
		tests = append(tests, ProwTest{
			Name:              tv.name,
			Branches:          branches,
			ClusterType:       clusterTypeForTest(cfg.Steps.Workflow, cfg.Steps.ClusterProfile, ocpVersionForBranches(branches)),
			Triggers:          triggersForProwTest(branches, cfg.Cron, cfg.Postsubmit),
			RunsAutomatically: cfg.AlwaysRun == nil || *cfg.AlwaysRun,
			Optional:          cfg.Optional,
			Description:       descriptionForTestFamily(testFamily(tv.name)),
			TestFamily:        testFamily(tv.name),
		})
	}
	return tests, nil
}

// prowTestConfigKey returns a key that's equal for two prowTestConfig values
// iff they'd produce the same ProwTest fields (aside from Name/Branches), so
// the same-named test defined identically across multiple branch/variant
// config files collapses into a single ProwTest spanning all those branches,
// while a genuinely different definition under the same name stays distinct.
func prowTestConfigKey(cfg prowTestConfig) string {
	alwaysRun := "default"
	if cfg.AlwaysRun != nil {
		alwaysRun = fmt.Sprintf("%v", *cfg.AlwaysRun)
	}
	return fmt.Sprintf("always_run=%s|optional=%v|postsubmit=%v|cron=%s|workflow=%s|cluster_profile=%s",
		alwaysRun, cfg.Optional, cfg.Postsubmit, cfg.Cron, cfg.Steps.Workflow, cfg.Steps.ClusterProfile)
}

// commonPrefix returns the longest string that's a prefix of every string in
// strs, mirroring Python's os.path.commonprefix (which openshift/release's
// config.py uses to derive each config file's "short name" by stripping it
// off the filename).
func commonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

// triggersForProwTest derives the trigger category from the actual
// ci-operator fields (cron:, steps.postsubmit:) rather than from job-naming
// conventions. A test defined only under "nightlies"-family branches is
// called out as TriggerNightly regardless of whether its underlying
// mechanism is a postsubmit (gated on a push to the nightlies branch) or a
// true periodic (cron:) -- every nightlies-only test in this repo's
// ci-operator config is one or the other, never both at once, and no test's
// branch set mixes nightlies with a non-nightlies branch, so this override
// never misclassifies a job that also runs elsewhere.
func triggersForProwTest(branches []string, cron string, postsubmit bool) []Trigger {
	if isNightlyOnly(branches) {
		return []Trigger{TriggerNightly}
	}
	switch {
	case cron != "":
		return []Trigger{TriggerPeriodic}
	case postsubmit:
		return []Trigger{TriggerPostMerge}
	default:
		return []Trigger{TriggerPreMerge}
	}
}

func isNightlyOnly(branches []string) bool {
	if len(branches) == 0 {
		return false
	}
	for _, branch := range branches {
		if !strings.HasPrefix(branch, "nightlies") {
			return false
		}
	}
	return true
}

var (
	ocpDashVersionPattern   = regexp.MustCompile(`ocp-(\d+)-(\d+)$`)
	ocpDottedVersionPattern = regexp.MustCompile(`ocp-(\d+\.\d+)`)
)

// ocpVersionForBranch extracts a human-readable OpenShift version label from
// a branch/variant column name such as "master__ocp-4-22", or "" if the
// branch has no variant suffix to read a version from (e.g. plain "master").
func ocpVersionForBranch(branch string) string {
	_, variant, found := strings.Cut(branch, "__")
	if !found {
		return ""
	}
	if m := ocpDashVersionPattern.FindStringSubmatch(variant); m != nil {
		return m[1] + "." + m[2]
	}
	if m := ocpDottedVersionPattern.FindStringSubmatch(variant); m != nil {
		return m[1]
	}
	return strings.TrimPrefix(variant, "ocp-")
}

func ocpVersionForBranches(branches []string) string {
	for _, branch := range branches {
		if v := ocpVersionForBranch(branch); v != "" {
			return v
		}
	}
	return ""
}

// clusterTypeForTest derives a human-readable cluster type from the
// ci-operator workflow (and, failing that, cluster_profile) a test uses.
// Workflow names in this repo's ci-operator config deterministically encode
// the cluster provider, so this is a lookup, not a guess.
func clusterTypeForTest(workflow, clusterProfile, ocpVersion string) string {
	switch {
	case strings.Contains(workflow, "rosa-hcp-e2e"):
		return "ROSA HCP (Red Hat OpenShift on AWS, hosted control plane)"
	case strings.Contains(workflow, "rosa-e2e"):
		return "ROSA (Red Hat OpenShift on AWS)"
	case strings.Contains(workflow, "aks-e2e"):
		return "AKS (Azure Kubernetes Service)"
	case strings.Contains(workflow, "aro-e2e"):
		return "ARO (Azure Red Hat OpenShift)"
	case strings.Contains(workflow, "osd-aws-e2e"):
		return "OpenShift Dedicated on AWS"
	case strings.Contains(workflow, "osd-gcp-e2e"):
		return "OpenShift Dedicated on GCP"
	case strings.Contains(workflow, "eks-e2e"):
		return "EKS (Amazon Elastic Kubernetes Service)"
	case strings.Contains(workflow, "ibmcloudz-e2e"):
		return "IBM Cloud on IBM Z"
	case strings.Contains(workflow, "powervs-e2e"):
		return "IBM Power Systems Virtual Server"
	case strings.Contains(workflow, "ocp-4-e2e"):
		return openShiftLabel(ocpVersion)
	case workflow == "stackrox-stackrox-e2e-job":
		return "GKE (Google Kubernetes Engine)"
	case strings.Contains(workflow, "firewatch"):
		return "OpenShift (Red Hat CSPI interop pipeline, AWS IPI)"
	case clusterProfile == "aws-stackrox" || clusterProfile == "aws-cspi-qe":
		// Catch-all for jobs (e.g. perf-scale jobs using the
		// stackrox-install-roxie workflow) that provision a self-managed
		// OpenShift-on-AWS cluster without using the standard
		// *-ocp-4-e2e workflow.
		return openShiftLabel(ocpVersion)
	default:
		return "Unknown"
	}
}

func openShiftLabel(ocpVersion string) string {
	if ocpVersion == "" {
		return "OpenShift (self-managed, AWS IPI)"
	}
	return fmt.Sprintf("OpenShift %s (self-managed, AWS IPI)", ocpVersion)
}

// familyPrefixes are cluster-provider/version/trigger-shape prefixes that get
// stripped (in order, repeatedly) from a Prow test name to find the
// underlying test suite it runs, e.g. "merge-gke-latest-qa-e2e-tests" and
// "rosa-fips-qa-e2e-tests" both reduce to "qa-e2e-tests". This list was
// hand-verified against every Prow test name found in tmp/prow-jobs.html.
var familyPrefixes = []string{
	"merge-",
	"daily-",
	"24nodes-",
	"gke-latest-",
	"gke-oldest-",
	"gke-race-condition-",
	"gke-external-pg-17-",
	"gke-",
	"aks-",
	"aro-",
	"eks-",
	"osd-aws-",
	"osd-gcp-",
	"rosa-hcp-",
	"rosa-fips-",
	"rosa-",
	"fips-",
	"ibmcloudz-",
	"powervs-",
	"ocp-",
	"latest-",
	"lts-",
	"stable-",
}

var versionedProviderPrefix = regexp.MustCompile(`^(ibmcloudz|powervs)-\d+(-\d+)?-`)

// acsInteropFamily hand-maps the small set of irregularly-named Red Hat
// CSPI/interop jobs, which don't fit the "<provider->-<suite>-tests" naming
// convention the other 83 Prow test names follow.
var acsInteropFamily = map[string]string{
	"cr--acs--tests-aws":  "acs-interop-tests",
	"acs-tests-aws":       "acs-interop-tests",
	"acs--tests-aws-fips": "acs-interop-tests",
	"acs-tests-aws-fips":  "acs-interop-tests",
}

// testFamily returns the normalized suite name used to group a test with
// every other test (Prow or GitHub Actions) that runs the same underlying
// suite against a different cluster provider or version.
func testFamily(name string) string {
	if family, ok := acsInteropFamily[name]; ok {
		return family
	}

	remaining := name
	for {
		if m := versionedProviderPrefix.FindString(remaining); m != "" {
			remaining = strings.TrimPrefix(remaining, m)
			continue
		}
		trimmedAny := false
		for _, prefix := range familyPrefixes {
			if strings.HasPrefix(remaining, prefix) {
				remaining = strings.TrimPrefix(remaining, prefix)
				trimmedAny = true
				break
			}
		}
		if !trimmedAny {
			break
		}
	}
	return strings.ReplaceAll(remaining, "corebpf-", "")
}

// familyDescriptions gives a 1-2 sentence, hand-written summary of what each
// known test family actually exercises. A family not listed here yields
// "Unknown" rather than a guessed description.
var familyDescriptions = map[string]string{
	"qa-e2e-tests":                  "Runs the main Groovy-based QA end-to-end test suite (qa-tests-backend) against a live cluster.",
	"ui-e2e-tests":                  "Runs the Cypress-based UI end-to-end test suite against a live cluster.",
	"operator-e2e-tests":            "Runs end-to-end tests of installing and operating StackRox via the Operator.",
	"nongroovy-e2e-tests":           "Runs the non-Groovy (bats-based) end-to-end test suite.",
	"vm-scanning-e2e-tests":         "Runs end-to-end tests of virtual machine vulnerability scanning.",
	"scanner-v4-install-tests":      "Runs end-to-end tests that verify Scanner V4 installs and functions correctly.",
	"compliance-e2e-tests":          "Runs end-to-end tests of the compliance scanning feature.",
	"nongroovy-compatibility-tests": "Runs the non-Groovy e2e suite to check compatibility across supported versions.",
	"version-compatibility-tests":   "Runs e2e tests to check compatibility across supported Kubernetes/platform versions.",
	"scale-tests":                   "Runs cluster scale tests to check StackRox behavior under a larger number of nodes/workloads.",
	"scale-test":                    "Runs a 24-node cluster scale/performance test.",
	"scale-scannerv4-test":          "Runs a 24-node cluster scale/performance test focused on Scanner V4.",
	"upgrade-tests":                 "Runs end-to-end tests of upgrading StackRox from a previous version.",
	"acs-interop-tests":             "Runs Red Hat CSPI/interop pipeline tests (QA e2e and compliance suites) against a managed OpenShift layered-product cluster.",
}

func descriptionForTestFamily(family string) string {
	if description, ok := familyDescriptions[family]; ok {
		return description
	}
	return "Unknown"
}
