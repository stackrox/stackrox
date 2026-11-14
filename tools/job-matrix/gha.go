package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// GHAEntry describes one GitHub Actions workflow file.
type GHAEntry struct {
	FileName     string
	WorkflowName string
	Triggers     []Trigger
	ClusterType  string
	Description  string
	TestFamily   string
	// RequiredLabel is set when a PR-triggered workflow only does anything
	// when a specific label is present (e.g. "if:
	// contains(github.event.pull_request.labels.*.name, '...')" gating the
	// whole job), as opposed to running unconditionally on every PR.
	RequiredLabel string
}

// ghaMeta is the hand-curated, per-file information that can't be read
// reliably off the workflow YAML's structured fields (what cluster the jobs
// provision, what they're actually testing, and which cross-cutting suite
// they belong to).
type ghaMeta struct {
	ClusterType   string
	Description   string
	TestFamily    string
	RequiredLabel string
}

// ghaMetadata maps workflow file name -> hand-curated metadata, gathered by
// reading every workflow under .github/workflows during the research for
// this tool. A workflow file not present here (added after this map was
// written) still appears in the output, with Unknown fields, rather than
// being silently dropped -- that drift is itself useful signal.
var ghaMetadata = map[string]ghaMeta{
	"e2e-dispatch.yaml": {
		ClusterType: "None (orchestrator; dispatches other e2e workflows)",
		Description: "Decides which downstream e2e workflows should run for a given push or PR, waits for images to publish, then triggers them.",
	},
	"e2e-qa-tests.yaml": {
		ClusterType: "GKE or OpenShift (configurable via cluster-provisioning-method/infra-flavor inputs)",
		Description: "Reusable workflow that deploys StackRox and runs the Groovy QA e2e suite (qa-tests-backend) against a live cluster.",
		TestFamily:  "qa-e2e-tests",
	},
	"e2e-qa-tests-gke.yaml": {
		ClusterType: "GKE (direct provisioning via scripts/ci/gke.sh)",
		Description: "Runs the QA e2e suite against a standard GKE cluster.",
		TestFamily:  "qa-e2e-tests",
	},
	"e2e-qa-tests-gke-konflux.yaml": {
		ClusterType: "GKE (direct provisioning via scripts/ci/gke.sh)",
		Description: "Runs the QA e2e suite against a GKE cluster using Konflux-built images.",
		TestFamily:  "qa-e2e-tests",
	},
	"e2e-qa-tests-ocp-fips-konflux.yaml": {
		ClusterType: "OpenShift 4 (FIPS-enabled, via stackrox/actions/infra)",
		Description: "Runs the QA e2e suite against a FIPS-enabled OpenShift 4 cluster using Konflux-built images.",
		TestFamily:  "qa-e2e-tests",
	},
	"e2e-nongroovy-tests.yaml": {
		ClusterType: "GKE (direct provisioning via scripts/ci/gke.sh)",
		Description: "Runs the non-Groovy (bats-based) end-to-end test suite.",
		TestFamily:  "nongroovy-e2e-tests",
	},
	"e2e-gke-upgrade-tests.yaml": {
		ClusterType: "GKE (direct provisioning via scripts/ci/gke.sh)",
		Description: "Runs StackRox upgrade tests (Central, Postgres major-version, Sensor) against a GKE cluster.",
		TestFamily:  "upgrade-tests",
	},
	"e2e-db-backup-restore-tests.yaml": {
		ClusterType: "GKE (direct provisioning via scripts/ci/gke.sh)",
		Description: "Reusable workflow that deploys StackRox and runs a Central database backup-and-restore test.",
		TestFamily:  "db-backup-restore-tests",
	},
	"e2e-db-backup-restore-tests-gke.yaml": {
		ClusterType: "GKE (direct provisioning via scripts/ci/gke.sh)",
		Description: "Runs the Central DB backup-and-restore test against a standard GKE cluster.",
		TestFamily:  "db-backup-restore-tests",
	},
	"e2e-db-backup-restore-tests-gke-konflux.yaml": {
		ClusterType: "GKE (direct provisioning via scripts/ci/gke.sh)",
		Description: "Runs the Central DB backup-and-restore test against a GKE cluster using Konflux-built images.",
		TestFamily:  "db-backup-restore-tests",
	},
	"e2e-byodb-tests.yaml": {
		ClusterType: "GKE (direct provisioning via scripts/ci/gke.sh)",
		Description: "Tests the Bring-Your-Own-Database (BYODB) deployment mode against a GKE cluster.",
		TestFamily:  "byodb-tests",
	},
	"scanner-e2e-test.yaml": {
		ClusterType:   "minikube",
		Description:   "Deploys Scanner V4 and Scanner V4 DB, then runs functional tests against the running scanner.",
		TestFamily:    "scanner-functional-tests",
		RequiredLabel: "scanner-functional-tests",
	},
	"emailsender-central-compatibility.yaml": {
		ClusterType: "KinD",
		Description: "Cross-repo compatibility test between stackrox/stackrox and stackrox/acs-fleet-manager for the email-sender/m2m-auth integration.",
	},
	"performance-tests.yml": {
		ClusterType:   "GKE (via stackrox/actions/infra)",
		Description:   "Launches Central, generates synthetic workload, and runs k6 load/performance tests.",
		TestFamily:    "performance-tests",
		RequiredLabel: "ci-performance-tests",
	},
	"unit-tests.yaml": {
		ClusterType: "None (KinD only for the sensor-integration-tests job)",
		Description: "Runs Go unit/integration tests, UI unit/component tests, operator Helm tests, shell unit tests, and a KinD-based sensor integration test.",
		TestFamily:  "unit-tests",
	},
	"build.yaml": {
		ClusterType: "None",
		Description: "Builds and pushes StackRox images, binaries, and artifacts (UI, CLI, main image, DB image, operator, scanner).",
	},
	"style.yaml": {
		ClusterType: "None",
		Description: "Runs linting and static style checks (Go, generated files, GitHub Actions, shellcheck).",
	},
	"check-crd-compatibility.yaml": {
		ClusterType: "None",
		Description: "Statically diffs CRD schemas against the previously released version to catch incompatible changes.",
	},
	"check-crd-frozen-on-release.yaml": {
		ClusterType: "None",
		Description: "Ensures CRD YAML files are unchanged on a release branch once that branch's first patch release has shipped.",
	},
	"scanner-db-integration-tests.yaml": {
		ClusterType: "None (local Postgres in-container)",
		Description: "Runs Scanner database integration tests against a local Postgres instance.",
	},
	"check-pr-title.yaml": {
		ClusterType: "None",
		Description: "Validates that the PR title follows the required format.",
	},
	"create-clusters.yml": {
		ClusterType: "GKE and/or OpenShift 4 (demo/long-running clusters via stackrox/actions)",
		Description: "Manually triggered release-process workflow that stands up demo/release clusters.",
	},
	"cut-rc.yml": {
		ClusterType: "GKE and/or OpenShift 4 (demo/long-running clusters via stackrox/actions)",
		Description: "Multi-stage automation for cutting a release candidate, including creating demo clusters.",
	},
	"upgrade-test.yml": {
		ClusterType: "GKE (via stackrox/actions/infra)",
		Description: "Creates GKE clusters to prepare a manual upgrade test between a previous release and a release candidate.",
	},
	"finish-release.yml": {
		ClusterType: "None",
		Description: "Finalizes a release: preflight checks, publishing the release, updating the infra demo version.",
	},
	"start-release.yml": {
		ClusterType: "None",
		Description: "Starts a new release: branch creation, CI triggering, changelog/version patching, milestone setup.",
	},
	"verify-release.yml": {
		ClusterType: "None",
		Description: "Verifies a published release (errata, version checks).",
	},
	"variables.yml": {
		ClusterType: "None",
		Description: "Reusable workflow that parses a version string into release/patch/rc/branch components for other release workflows.",
	},
	"scan-image-vulnerabilities.yaml": {
		ClusterType: "None",
		Description: "Scans built release images for vulnerabilities.",
	},
	"add-vms-to-cluster.yml": {
		ClusterType: "Existing infra cluster (adds VMs to it; does not create a cluster)",
		Description: "Ensures a given number of RHEL VMs exist on a named infra cluster.",
	},
	"release-ci.yaml": {
		ClusterType: "None",
		Description: "Release-branch packaging/publishing pipeline (build, version checks, Helm/CLI/OpenAPI publishing).",
	},
	"tagged-ci.yaml": {
		ClusterType: "None",
		Description: "Posts a Slack notification when a release tag is pushed.",
	},
	"nightly-tag.yml": {
		ClusterType: "None",
		Description: "Creates a nightly build tag.",
	},
	"scanner-mapping-update.yaml": {
		ClusterType: "None",
		Description: "Scheduled job that updates Scanner's vulnerability mapping data.",
	},
	"scanner-nvd-update.yaml": {
		ClusterType:   "None",
		Description:   "Scheduled job that fetches and uploads NVD vulnerability feed data for Scanner.",
		RequiredLabel: "pr-update-scanner-nvd",
	},
	"scanner-offline-bundle-update.yaml": {
		ClusterType:   "None",
		Description:   "Scheduled job that generates and uploads the Scanner offline vulnerability bundle.",
		RequiredLabel: "pr-update-scanner-vulns",
	},
	"scanner-updater-bundle.yaml": {
		ClusterType: "None",
		Description: "Scheduled job that builds and publishes the Scanner updater bundle.",
	},
	"scanner-updater-export.yaml": {
		ClusterType: "None",
		Description: "Reusable workflow that exports/updates Scanner updater data.",
	},
	"scanner-updater-schedule.yaml": {
		ClusterType:   "None",
		Description:   "Scheduled job that plans and runs Scanner updater updates.",
		RequiredLabel: "pr-update-scanner-vulns",
	},
	"scanner-versioned-definitions-update.yaml": {
		ClusterType:   "None",
		Description:   "Builds and uploads versioned Scanner vulnerability definitions.",
		RequiredLabel: "pr-update-scanner-vulns",
	},
	"signing-key-bundle-update.yaml": {
		ClusterType: "None",
		Description: "Validates and uploads the Red Hat signing key bundle.",
	},
	"add-new-pr-to-oss-triaging.yml": {
		ClusterType: "None",
		Description: "Adds new external-contributor PRs to the OSS triaging project board.",
	},
	"audit-backport-prs.yml": {
		ClusterType: "None",
		Description: "Audits backport PR hygiene and posts results to Slack.",
	},
	"auto-merge.yml": {
		ClusterType: "None",
		Description: "Automatically merges eligible pull requests on a schedule.",
	},
	"backport-pr.yaml": {
		ClusterType:   "None",
		Description:   "Opens a backport PR to a release branch when a merged PR is labeled for backport.",
		RequiredLabel: "backport release-<version>",
	},
	"batch-load-test-metrics.yml": {
		ClusterType: "None",
		Description: "Unknown",
	},
	"ci-failures-report.yml": {
		ClusterType: "None",
		Description: "Posts a report of CI failure statistics to Slack.",
	},
	"detect-changes.yml": {
		ClusterType: "None",
		Description: "Reusable workflow that computes which files changed, so other workflows can skip unnecessary jobs.",
	},
	"fixxxer.yaml": {
		ClusterType: "None",
		Description: "Applies automated fixes to a PR in response to a comment command.",
	},
	"konflux-auto-retest.yml": {
		ClusterType: "None",
		Description: "Periodically retests failed Konflux builds.",
	},
	"labeler.yaml": {
		ClusterType: "None",
		Description: "Labels pull requests based on which files changed.",
	},
	"mintmaker-auto-merge.yml": {
		ClusterType:   "None",
		Description:   "Auto-approves and merges Konflux Mintmaker dependency-update PRs.",
		RequiredLabel: "mintmaker-auto-merge",
	},
	"notify-milestone-change.yml": {
		ClusterType: "None",
		Description: "Posts a Slack notification when a PR's milestone changes.",
	},
	"retest_comment.yml": {
		ClusterType: "None",
		Description: "Handles the /retest PR-comment command to rerun failed checks.",
	},
	"retest_periodic.yml": {
		ClusterType: "None",
		Description: "Periodically auto-retries failing CI checks.",
	},
	"tag-more-reviewers.yaml": {
		ClusterType: "None",
		Description: "Adds Konflux maintainers as reviewers on pull requests.",
	},
	"triage-report_periodic.yml": {
		ClusterType: "None",
		Description: "Generates a periodic support triage report.",
	},
	"update_collector_periodic.yaml": {
		ClusterType: "None",
		Description: "Opens a PR bumping the collector dependency version.",
	},
	"update_fact_periodic.yaml": {
		ClusterType: "None",
		Description: "Opens a PR bumping the fact dependency version.",
	},
}

// ghaTriggerNames maps a GitHub Actions `on:` event name to our shared
// Trigger enum. Event names with no clean equivalent (e.g. issue_comment,
// milestone) are intentionally left unmapped.
var ghaTriggerNames = map[string]Trigger{
	"pull_request":        TriggerPreMerge,
	"pull_request_target": TriggerPreMerge,
	"push":                TriggerPostMerge,
	"schedule":            TriggerPeriodic,
	"workflow_dispatch":   TriggerManual,
	"workflow_call":       TriggerReusable,
}

type workflowFile struct {
	Name string      `yaml:"name"`
	On   interface{} `yaml:"on"`
}

// GHAWorkflows discovers every *.yml/*.yaml file directly under dir, reads
// its `on:` trigger configuration, and attaches the hand-curated metadata
// for cluster type / description / test family.
func GHAWorkflows(dir string) ([]GHAEntry, error) {
	var files []string
	for _, pattern := range []string{"*.yml", "*.yaml"} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, fmt.Errorf("globbing %s in %s: %w", pattern, dir, err)
		}
		files = append(files, matches...)
	}
	sort.Strings(files)

	entries := make([]GHAEntry, 0, len(files))
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		var wf workflowFile
		if err := yaml.Unmarshal(data, &wf); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}

		fileName := filepath.Base(path)
		meta, ok := ghaMetadata[fileName]
		if !ok {
			meta = ghaMeta{ClusterType: "Unknown", Description: "Unknown"}
		}

		entries = append(entries, GHAEntry{
			FileName:      fileName,
			WorkflowName:  wf.Name,
			Triggers:      triggersFromOn(wf.On),
			ClusterType:   meta.ClusterType,
			Description:   meta.Description,
			TestFamily:    meta.TestFamily,
			RequiredLabel: meta.RequiredLabel,
		})
	}
	return entries, nil
}

func triggersFromOn(on interface{}) []Trigger {
	var triggers []Trigger
	seen := map[Trigger]bool{}
	for _, name := range triggerEventNames(on) {
		trigger, ok := ghaTriggerNames[name]
		if !ok || seen[trigger] {
			continue
		}
		seen[trigger] = true
		triggers = append(triggers, trigger)
	}
	if len(triggers) == 0 {
		return []Trigger{TriggerUnknown}
	}
	return triggers
}

// triggerEventNames normalizes the three shapes a workflow's `on:` value can
// take -- a single string, a list of strings, or a map keyed by event name
// -- into a plain list of event names.
func triggerEventNames(on interface{}) []string {
	switch v := on.(type) {
	case string:
		return []string{v}
	case []interface{}:
		names := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				names = append(names, s)
			}
		}
		return names
	case map[string]interface{}:
		names := make([]string, 0, len(v))
		for k := range v {
			names = append(names, k)
		}
		sort.Strings(names)
		return names
	default:
		return nil
	}
}
