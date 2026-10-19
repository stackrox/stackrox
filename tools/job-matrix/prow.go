package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
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

// htmlFileProwSource reads Prow test definitions out of a saved copy of the
// ci-operator "test matrix" HTML page for stackrox/stackrox (the page
// rendered at the config-summary anchor), rather than talking to Prow live.
type htmlFileProwSource struct {
	path string
}

// NewHTMLFileProwSource returns a ProwSource that reads ci-operator test
// definitions from the ci-operator test-matrix HTML export at path.
func NewHTMLFileProwSource(path string) ProwSource {
	return htmlFileProwSource{path: path}
}

// prowTestConfig mirrors the subset of the ci-operator test schema that's
// embedded in each config-summary table cell.
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

func (s htmlFileProwSource) Tests() ([]ProwTest, error) {
	f, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", s.path, err)
	}
	defer f.Close()

	doc, err := html.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parsing %s as HTML: %w", s.path, err)
	}

	ordered := documentOrder(doc)

	configSummaryHeading := findByID(ordered, "config-summary")
	if configSummaryHeading == nil {
		return nil, fmt.Errorf("%s: could not find a #config-summary section", s.path)
	}
	table := findNextElement(ordered, configSummaryHeading, "table")
	if table == nil {
		return nil, fmt.Errorf("%s: could not find the config summary table", s.path)
	}

	variants, rows, err := configSummaryVariantsAndRows(table)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.path, err)
	}

	type testVariant struct {
		name string
		yaml string
	}
	var order []testVariant
	branchesFor := map[testVariant][]string{}

	for _, row := range rows {
		cells := childElements(row, "td")
		if len(cells) == 0 {
			continue
		}
		name := strings.TrimSpace(textContent(cells[0]))
		if name == "" {
			continue
		}
		for i, cell := range cells[1:] {
			pres := descendantElements(cell, "pre")
			if len(pres) == 0 {
				continue
			}
			variantIdx := i + 1
			if variantIdx >= len(variants) {
				continue
			}
			tv := testVariant{name: name, yaml: textContent(pres[0])}
			if _, seen := branchesFor[tv]; !seen {
				order = append(order, tv)
			}
			branchesFor[tv] = append(branchesFor[tv], variants[variantIdx])
		}
	}

	tests := make([]ProwTest, 0, len(order))
	for _, tv := range order {
		var cfg prowTestConfig
		if err := yaml.Unmarshal([]byte(tv.yaml), &cfg); err != nil {
			return nil, fmt.Errorf("%s: parsing ci-operator config for test %q: %w", s.path, tv.name, err)
		}
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

// configSummaryVariantsAndRows reads the header's branch/variant column
// names and the <tr> rows that make up the "tests" section of the config
// summary table (skipping the base-image rows that precede it).
func configSummaryVariantsAndRows(table *html.Node) ([]string, []*html.Node, error) {
	theads := childElements(table, "thead")
	tbodies := childElements(table, "tbody")
	if len(theads) == 0 || len(tbodies) == 0 {
		return nil, nil, fmt.Errorf("config summary table is missing a thead or tbody")
	}

	headerRows := childElements(theads[0], "tr")
	if len(headerRows) == 0 {
		return nil, nil, fmt.Errorf("config summary table has no header row")
	}
	headerCells := childElements(headerRows[0], "th")
	variants := make([]string, len(headerCells))
	for i, th := range headerCells {
		variants[i] = strings.TrimSpace(textContent(th))
	}

	rows := childElements(tbodies[0], "tr")
	testsHeaderIdx := -1
	for i, row := range rows {
		cells := childElements(row, "td")
		if len(cells) > 0 && strings.TrimSpace(textContent(cells[0])) == "tests" {
			testsHeaderIdx = i
			break
		}
	}
	if testsHeaderIdx == -1 {
		return nil, nil, fmt.Errorf("could not find the \"tests\" section of the config summary table")
	}

	return variants, rows[testsHeaderIdx+1:], nil
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

// --- small html.Node helpers (golang.org/x/net/html has no query API) ---

func documentOrder(root *html.Node) []*html.Node {
	var nodes []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		nodes = append(nodes, n)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return nodes
}

func findByID(nodes []*html.Node, id string) *html.Node {
	for _, n := range nodes {
		if n.Type != html.ElementNode {
			continue
		}
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return n
			}
		}
	}
	return nil
}

// findNextElement returns the first element with the given tag that appears
// after `after` in document order.
func findNextElement(nodes []*html.Node, after *html.Node, tag string) *html.Node {
	passedAfter := false
	for _, n := range nodes {
		if n == after {
			passedAfter = true
			continue
		}
		if passedAfter && n.Type == html.ElementNode && n.Data == tag {
			return n
		}
	}
	return nil
}

func childElements(n *html.Node, tag string) []*html.Node {
	var out []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			out = append(out, c)
		}
	}
	return out
}

func descendantElements(n *html.Node, tag string) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == tag {
				out = append(out, c)
			}
			walk(c)
		}
	}
	walk(n)
	return out
}

func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}
