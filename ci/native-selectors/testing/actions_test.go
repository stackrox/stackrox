package testing_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComparisonScenarios(t *testing.T) {
	f := newFixture(t)
	var comparison struct {
		SHA      string
		Fixtures map[string]string
	}
	f.json(t, "testing/comparison.json", &comparison)
	require.Equal(t, "34873a53e3b60b5fc7b50b1446b02d11a1eb0fca", comparison.SHA)
	for file, hash := range comparison.Fixtures {
		require.Equal(t, hash, digest(f.read(t, "ci/native-selectors/testing/fixtures/"+file)), "pinned fixture %s", file)
	}
	require.Equal(t, 29, len(f.local)+len(f.consumers))
	var scenarios []struct {
		Name          string
		Files, Labels []string
		Enforced      bool
		options
	}
	var expected map[string]struct {
		Selected []string
		Dispatch struct {
			Jobs     string
			Ordinary string `json:"wait-for-gha-images"`
			Konflux  string `json:"wait-for-konflux-images"`
		}
	}
	f.json(t, "testing/fixtures/scenarios.json", &scenarios)
	f.json(t, "testing/fixtures/expected.json", &expected)
	require.Len(t, scenarios, 83)
	// additions records intentional departures without modifying the pinned evidence.
	additions := map[string][]string{
		"makefile":              {"go", "go-postgres", "go-bench", "sensor-integration-tests", "ui", "ui-component", "local-roxctl-tests", "shell-unit-tests"},
		"CI workflow":           {"go", "go-postgres", "go-bench", "sensor-integration-tests", "ui", "ui-component", "local-roxctl-tests", "shell-unit-tests"},
		"mixed CI and product":  {"go-postgres", "sensor-integration-tests", "ui", "ui-component", "local-roxctl-tests", "shell-unit-tests", "check-generated-files"},
		"UI Go file":            {"check-generated-files"},
		"scanner":               {"check-generated-files"},
		"scanner and docs":      {"check-generated-files"},
		"operator":              {"check-generated-files"},
		"roxctl":                {"check-generated-files"},
		"central only":          {"check-generated-files"},
		"sensor only":           {"check-generated-files"},
		"mixed central sensor":  {"check-generated-files"},
		"mixed UI central":      {"check-generated-files"},
		"shared":                {"check-generated-files"},
		"central communication": {"check-generated-files"},
		"sensor communication":  {"check-generated-files"},
		"legacy selector files": {"check-generated-files"},
	}
	compared := 0
	departures := 0
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			labels := slices.Clone(scenario.Labels)
			if scenario.Enforced {
				labels = append(labels, "ci-dispatcher-enforce")
			}
			contexts := map[string]object{}
			context := func(workflow string) object {
				input := str(lookup(f.jobs(workflow), "detect-changes", "with", "run_all"))
				if ctx, ok := contexts[input]; ok {
					return ctx
				}
				opt := scenario.options
				opt.workflow = workflow
				ctx := f.context(t, scenario.Files, labels, opt)
				contexts[input] = ctx
				return ctx
			}
			want, ok := expected[scenario.Name]
			require.True(t, ok)
			if scenario.Event == "pull_request" {
				targets := slices.Clone(f.consumers)
				for name := range f.local {
					targets = append(targets, name)
				}
				for _, target := range targets {
					workflow := "e2e-dispatch.yaml"
					if selected, ok := f.local[target]; ok {
						workflow = selected.workflow
					}
					baseline := slices.Contains(want.Selected, target)
					additional := slices.Contains(additions[scenario.Name], target)
					if additional {
						require.False(t, baseline, "departure must add a candidate: %s", target)
						departures++
					}
					require.Equal(t, baseline || additional, f.candidate(t, target, context(workflow)), "candidate %s", target)
					compared++
				}
			}
			jobs, ordinary, konflux := f.dispatch(t, context("e2e-dispatch.yaml"))
			var expectedJobs map[string]bool
			require.NoError(t, json.Unmarshal([]byte(want.Dispatch.Jobs), &expectedJobs))
			require.Equal(t, expectedJobs, jobs)
			require.Equal(t, want.Dispatch.Ordinary == "true", ordinary)
			require.Equal(t, want.Dispatch.Konflux == "true", konflux)
			for name, runs := range jobs {
				if runs && scenario.Enforced {
					if strings.Contains(name, "konflux") {
						require.True(t, konflux, "image prerequisite for %s", name)
					} else {
						require.True(t, ordinary, "image prerequisite for %s", name)
					}
				}
			}
		})
	}
	require.Equal(t, 2175, compared)
	require.Equal(t, 36, departures)
	t.Logf("%d candidate comparisons across 29 targets: %d unchanged, %d intentional additions; %d dispatch/waiter scenarios", compared, compared-departures, departures, len(scenarios))
}

func TestSelectionInteractions(t *testing.T) {
	f := newFixture(t)
	enforced := []string{"ci-dispatcher-enforce"}
	for name, test := range map[string]struct {
		labels []string
		run    bool
	}{
		"enforcement skips unrelated PostgreSQL": {labels: enforced},
		"shadow permits unrelated PostgreSQL":    {run: true},
		"area label selects PostgreSQL":          {labels: []string{"ci-dispatcher-enforce", "area/central"}, run: true},
		"removing enforcement restores shadow":   {labels: []string{"area/central"}, run: true},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.run, f.explain(t, "go-postgres", f.context(t, []string{"README.md"}, test.labels, options{})).run)
		})
	}
	t.Run("labels cannot bypass outer guards", func(t *testing.T) {
		ctx := f.context(t, []string{"ui/package.json"}, append(slices.Clone(enforced), "ci-run-all-tests"), options{})
		require.True(t, f.candidate(t, "sensor-integration-tests", ctx))
		require.Equal(t, "false", evaluate(t, obj(f.jobs("unit-tests.yaml")["sensor-integration-tests"])["if"], ctx))
		ctx = f.context(t, []string{"README.md"}, append(slices.Clone(enforced), "area/ui"), options{})
		require.True(t, f.candidate(t, "ui", ctx))
		require.Equal(t, "false", evaluate(t, obj(f.jobs("unit-tests.yaml")["ui"])["if"], ctx))
	})
	t.Run("E2E label additions and removals use live labels", func(t *testing.T) {
		ctx := f.context(t, []string{"README.md"}, nil, options{liveLabels: []string{"ci-dispatcher-enforce", "e2e-byodb-tests"}})
		jobs, _, _ := f.dispatch(t, ctx)
		require.True(t, jobs["e2e-byodb-tests"])
		ctx = f.context(t, []string{"README.md"}, []string{"ci-dispatcher-enforce", "e2e-byodb-tests"}, options{liveLabels: enforced})
		jobs, _, _ = f.dispatch(t, ctx)
		require.False(t, jobs["e2e-byodb-tests"])
	})
	t.Run("failed label refresh disables enforcement", func(t *testing.T) {
		ctx := f.context(t, []string{"README.md"}, []string{"ci-dispatcher-enforce", "ci-run-all-tests"}, options{labelFailure: true})
		require.Equal(t, "false", lookup(ctx, "needs", "should-dispatch", "outputs", "enforced"))
		jobs, _, _ := f.dispatch(t, ctx)
		require.False(t, jobs["e2e-byodb-tests"])
	})
	t.Run("unknown diff defaults precede run-all", func(t *testing.T) {
		ctx := f.context(t, []string{"README.md"}, []string{"ci-dispatcher-enforce", "ci-run-all-tests"}, options{failure: true})
		jobs, _, _ := f.dispatch(t, ctx)
		require.True(t, jobs["e2e-byodb-tests"])
		require.False(t, jobs["e2e-qa-tests-gke"])
	})
	for _, event := range []string{"push", "workflow_dispatch"} {
		t.Run(event+" retains execution", func(t *testing.T) {
			ctx := f.context(t, nil, nil, options{Event: event})
			delete(obj(lookup(ctx, "github", "event")), "pull_request")
			obj(ctx["steps"])["labels"] = object{"outcome": "skipped", "outputs": object{}}
			f.dispatchOutputs(t, ctx)
			jobs, ordinary, konflux := f.dispatch(t, ctx)
			for name, run := range jobs {
				require.True(t, run, name)
			}
			require.True(t, ordinary)
			require.True(t, konflux)
		})
	}
	t.Run("ordinary reruns retain event labels", func(t *testing.T) {
		ctx := f.context(t, []string{"README.md"}, []string{"ci-dispatcher-enforce", "area/central"}, options{liveLabels: enforced})
		require.True(t, f.explain(t, "go-postgres", ctx).candidate)
	})
	t.Run("missing reporter output permits image prerequisites", func(t *testing.T) {
		ctx := f.context(t, []string{"go.mod"}, []string{"ci-dispatcher-enforce", "konflux-build"}, options{})
		obj(ctx["steps"])["selection"] = object{"outputs": object{}}
		for _, name := range []string{"wait-for-images", "wait-for-konflux-images"} {
			require.Equal(t, "true", evaluate(t, lookup(f.jobs("e2e-dispatch.yaml"), name, "outputs", "run"), ctx))
		}
		for name := range f.local {
			require.True(t, f.explain(t, name, ctx).candidate, "broad dependency: %s", name)
		}
	})
}

func TestExplicitOverrides(t *testing.T) {
	f := newFixture(t)
	labels := []string{"ci-dispatcher-enforce"}
	for name, paths := range map[string][]string{
		"Makefile":      {"Makefile", "README.md"},
		"make includes": {"make/gotools.mk", "README.md"},
		"unit workflow": {".github/workflows/unit-tests.yaml", "README.md"},
		"job preamble":  {".github/actions/job-preamble/action.yml", "README.md"},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := f.context(t, paths, labels, options{workflow: "unit-tests.yaml"})
			require.Equal(t, "true", lookup(ctx, "needs", "detect-changes", "outputs", "run_all"))
			for target, selected := range f.local {
				if selected.workflow == "unit-tests.yaml" {
					require.True(t, f.explain(t, target, ctx).run, target)
				}
			}
			ctx = f.context(t, paths, labels, options{workflow: "e2e-dispatch.yaml"})
			require.Empty(t, lookup(ctx, "needs", "detect-changes", "outputs", "run_all"), "overrides belong to the calling workflow")
		})
	}
	t.Run("detector changes select every candidate", func(t *testing.T) {
		for workflow := range f.workflows {
			ctx := f.context(t, []string{".github/workflows/detect-changes.yml", "README.md"}, labels, options{workflow: workflow})
			for target, selected := range f.local {
				if selected.workflow == workflow {
					require.True(t, f.candidate(t, target, ctx), target)
				}
			}
			if workflow == "e2e-dispatch.yaml" {
				for _, target := range f.consumers {
					require.True(t, f.candidate(t, target, ctx), target)
				}
			}
		}
	})
	t.Run("empty and unrelated diffs do not imply an override", func(t *testing.T) {
		for _, paths := range [][]string{nil, {"OWNERS"}} {
			ctx := f.context(t, paths, labels, options{})
			require.False(t, f.explain(t, "go-postgres", ctx).run)
		}
	})
	t.Run("classification failure retains per-job defaults", func(t *testing.T) {
		ctx := f.context(t, []string{".github/workflows/detect-changes.yml"}, labels, options{Unknown: true, workflow: "e2e-dispatch.yaml"})
		require.Equal(t, "true", lookup(ctx, "needs", "detect-changes", "outputs", "run_all"))
		jobs, _, _ := f.dispatch(t, ctx)
		require.True(t, jobs["e2e-byodb-tests"])
		require.False(t, jobs["e2e-qa-tests-gke"])
		require.False(t, jobs["e2e-qa-tests-gke-konflux"])
		require.False(t, jobs["e2e-qa-tests-ocp-fips-konflux"])
	})
	t.Run("overrides do not authorize forks or supply Konflux images", func(t *testing.T) {
		for _, fork := range []bool{false, true} {
			ctx := f.context(t, []string{".github/workflows/detect-changes.yml"}, labels, options{Fork: fork, Draft: true, workflow: "e2e-dispatch.yaml"})
			jobs, _, konflux := f.dispatch(t, ctx)
			require.Equal(t, !fork, jobs["e2e-byodb-tests"])
			require.False(t, konflux)
			require.False(t, jobs["e2e-qa-tests-gke-konflux"])
		}
	})
	t.Run("Go sources select generated-file verification", func(t *testing.T) {
		ctx := f.context(t, []string{"pkg/concurrency/context.go", "README.md"}, labels, options{workflow: "style.yaml"})
		require.True(t, f.explain(t, "check-generated-files", ctx).run)
	})
}
