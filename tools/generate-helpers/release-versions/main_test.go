package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func repository(t *testing.T) (string, func(...string), func(int, string)) {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
	}
	git("init", "-q", "--initial-branch=master")
	git("config", "core.hooksPath", "/dev/null")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	git("config", "commit.gpgsign", "false")
	git("config", "tag.gpgsign", "false")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "pkg/migrations/internal"), 0755))
	write := func(sequence int, tag string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(dir, sequencePath), []byte(fmt.Sprintf("package internal\nvar CurrentDBVersionSeqNum int = %d\n", sequence)), 0644))
		git("add", sequencePath)
		git("commit", "-qm", "sequence", "--allow-empty")
		if tag != "" {
			git("tag", tag)
		}
	}
	return dir, git, write
}

func TestDevelopmentBaseline(t *testing.T) {
	dir, git, write := repository(t)
	write(220, "4.10.0")
	write(225, "4.11.0")
	write(226, "5.0.0")
	write(227, "5.1.x")
	before, err := generate(dir, "5.1.x-105-g66fd4ac964")
	require.NoError(t, err)
	require.Contains(t, string(before), `Version: "5.1", Sequence: 227`)
	write(228, "")
	before, err = generate(dir, "5.1.x-106-gabcdef-dirty")
	require.NoError(t, err)
	require.Contains(t, string(before), `Version: "5.1", Sequence: 228`)
	git("tag", "5.1.0")
	after, err := generate(dir, "5.1.0")
	require.NoError(t, err)
	require.Equal(t, before, after)
	write(229, "5.1.1")
	after, err = generate(dir, "5.1.1")
	require.NoError(t, err)
	require.Equal(t, before, after)
	write(240, "5.2.0")
	after, err = generate(dir, "5.1.1")
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestHistoricalPatchMigration(t *testing.T) {
	dir, _, write := repository(t)
	write(205, "4.5.0")
	write(206, "4.5.1")
	write(209, "4.6.0")
	write(209, "4.7.0")
	write(211, "4.8.x")
	data, err := generate(dir, "4.8.x")
	require.NoError(t, err)
	require.Contains(t, string(data), `Version: "4.5", Sequence: 205`)
	require.NotContains(t, string(data), "206")
}

func TestGenerationFailures(t *testing.T) {
	for name, tc := range map[string]struct {
		target, tag, source, want string
		completeHistory           bool
	}{
		"missing floor":                  {target: "5.1.x", want: "4.10.0"},
		"RC is not GA":                   {target: "5.1.x", tag: "4.10.0-rc.1", want: "4.10.0"},
		"missing patch baseline":         {target: "5.1.1", tag: "4.10.0", want: "5.1.0", completeHistory: true},
		"patch RC missing baseline":      {target: "5.1.1-rc.0", tag: "4.10.0", want: "5.1.0", completeHistory: true},
		"patch nightly missing baseline": {target: "5.1.1-nightly-20260930", tag: "4.10.0", want: "5.1.0", completeHistory: true},
		"invalid target":                 {target: "garbage", want: "version"},
		"phantom target":                 {target: "4.12.x", want: "stream"},
		"malformed sequence":             {target: "5.1.x", tag: "4.10.0", source: "package internal\nvar CurrentDBVersionSeqNum = 0", want: "positive"},
	} {
		t.Run(name, func(t *testing.T) {
			dir, _, write := repository(t)
			write(220, tc.tag)
			if tc.completeHistory {
				write(225, "4.11.0")
				write(226, "5.0.0")
			}
			if tc.source != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, sequencePath), []byte(tc.source), 0644))
			}
			_, err := generate(dir, tc.target)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestPendingStreams(t *testing.T) {
	for name, tc := range map[string]struct {
		floor, latest, pending, target string
		withRC                         bool
	}{
		"major boundary RC-only": {"4.10", "4.11", "5.0", "5.1.x", true},
		"future development":     {"5.3", "5.4", "5.5", "5.6.x", true},
		"RC target":              {"5.3", "5.4", "5.5", "5.6.0-rc.1", true},
		"no RC tags":             {"5.3", "5.4", "5.5", "5.6.x", false},
	} {
		t.Run(name, func(t *testing.T) {
			dir, _, write := repository(t)
			write(220, tc.floor+".0")
			write(225, tc.latest+".0")
			if tc.withRC {
				write(0, tc.pending+".0-rc.1") // Historical RC sequences must not be read.
			}
			write(227, "")
			data, err := generate(dir, tc.target)
			require.NoError(t, err)
			require.Contains(t, string(data), fmt.Sprintf(`Version: %q, Sequence: 220`, tc.floor))
			require.Contains(t, string(data), fmt.Sprintf(`Version: %q, Sequence: 225`, tc.latest))
			require.NotContains(t, string(data), fmt.Sprintf(`Version: %q`, tc.pending))
			require.Contains(t, string(data), fmt.Sprintf(`Version: %q, Sequence: 227`, strings.Join(strings.Split(tc.target, ".")[:2], ".")))
		})
	}
}

func TestMissingIntermediateGA(t *testing.T) {
	for name, rc := range map[string]bool{"no tags": false, "old RC tags": true} {
		t.Run(name, func(t *testing.T) {
			dir, _, write := repository(t)
			write(220, "5.3.0")
			if rc {
				write(221, "5.4.0-rc.1")
			}
			write(225, "5.5.0")
			write(227, "")
			_, err := generate(dir, "5.6.x")
			require.ErrorContains(t, err, "missing initial GA tag 5.4.0")
		})
	}
}

func TestPendingStreamLifecycle(t *testing.T) {
	dir, _, write := repository(t)
	write(220, "5.3.0")
	write(225, "5.4.0")
	write(226, "5.5.0-rc.1")
	write(230, "")
	before, err := generate(dir, "5.6.x")
	require.NoError(t, err)
	require.NotContains(t, string(before), `Version: "5.5"`)
	write(228, "5.5.0")
	write(230, "")
	baseline, err := generate(dir, "5.6.x")
	require.NoError(t, err)
	require.Contains(t, string(baseline), `Version: "5.5", Sequence: 228`)
	require.Contains(t, string(baseline), `Version: "5.6", Sequence: 230`)
	for _, tag := range []string{"5.5.1-rc.1", "5.5.1"} {
		write(229, tag)
		write(230, "")
		after, err := generate(dir, "5.6.x")
		require.NoError(t, err)
		require.Equal(t, baseline, after)
	}
}

func TestGenerationCommand(t *testing.T) {
	dir, git, write := repository(t)
	write(220, "4.10.0")
	write(225, "4.11.0")
	write(226, "5.0.0")
	write(227, "5.1.x")
	versionMakefile(t, dir)
	t.Chdir(dir)
	t.Setenv("BUILD_TAG", "")
	require.NoError(t, run(""))
	output := filepath.Join(dir, "pkg/migrations/release_versions.go")
	before, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(before), `Version: "5.1", Sequence: 227`)
	git("tag", "5.1.0")
	t.Setenv("BUILD_TAG", "5.1.0")
	require.NoError(t, run(""))
	after, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, before, after)
	t.Setenv("BUILD_TAG", "5.5.x")
	require.ErrorContains(t, run(""), "5.2.0")
	after, err = os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, before, after, "failed generation must not overwrite the output")
	require.NoError(t, run("5.1.0"), "explicit target takes precedence over BUILD_TAG")
}

func versionMakefile(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "make"), 0755))
	for _, name := range []string{"env.mk", "goproxy.mk"} {
		data, err := os.ReadFile(filepath.Join("../../../make", name))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "make", name), data, 0644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte("include make/env.mk\ntag:\n\t@echo $(TAG)\n"), 0644))
}

func TestReleaseBranchLifecycle(t *testing.T) {
	dir, git, write := repository(t)
	versionMakefile(t, dir)
	t.Chdir(dir)
	t.Setenv("BUILD_TAG", "")
	write(220, "4.10.0")
	write(225, "4.11.0")
	write(227, "5.0.0")
	git("tag", "-a", "5.1.x", "-m", "development")
	write(228, "")
	git("checkout", "-qb", "release-5.1")
	write(228, "")
	git("tag", "-a", "5.1.0-rc.0", "-m", "release candidate")
	git("checkout", "master")
	write(230, "")
	git("tag", "-a", "5.2.x", "-m", "next development stream")
	git("checkout", "release-5.1")
	write(229, "")
	git("tag", "-a", "5.1.0-rc.1", "-m", "release candidate with migration")

	readOutput := func() []byte {
		t.Helper()
		require.NoError(t, run(""))
		data, err := os.ReadFile("pkg/migrations/release_versions.go")
		require.NoError(t, err)
		return data
	}
	before := readOutput()
	require.Contains(t, string(before), `Version: "5.1", Sequence: 229`)
	require.NotContains(t, string(before), `Version: "5.2"`)
	git("tag", "-a", "5.1.0", "-m", "GA")
	git("branch", "5.1.0", "master") // A branch name must not shadow the GA tag.
	require.Equal(t, before, readOutput())
	git("checkout", "--detach", "refs/tags/5.1.0")
	require.Equal(t, before, readOutput(), "detached release CI checkout")
	git("checkout", "release-5.1")
	write(231, "")
	git("tag", "-a", "5.1.1-rc.0", "-m", "patch migration")
	require.Equal(t, before, readOutput())
	git("tag", "-a", "5.1.1", "-m", "patch GA")
	require.Equal(t, before, readOutput())

	git("checkout", "master")
	write(240, "")
	git("tag", "-a", "5.2.0", "-m", "newer GA")
	after := readOutput()
	require.Contains(t, string(after), `Version: "5.1", Sequence: 229`, "GA on another branch is authoritative")
	require.Contains(t, string(after), `Version: "5.2", Sequence: 240`)
	git("checkout", "release-5.1")
	require.Equal(t, before, readOutput(), "newer release tags must not change old branch output")
}

func TestMultipleReleaseBranchesWithRCs(t *testing.T) {
	dir, git, write := repository(t)
	write(220, "4.10.0")
	git("checkout", "-qb", "release-4.10")
	git("checkout", "master")
	write(225, "4.11.0")
	git("checkout", "-qb", "release-4.11")
	git("checkout", "master")
	write(228, "5.1.x")

	before, err := generate(dir, "5.1.x")
	require.NoError(t, err)
	require.Contains(t, string(before), `Version: "4.10", Sequence: 220`)
	require.Contains(t, string(before), `Version: "4.11", Sequence: 225`)
	require.Contains(t, string(before), `Version: "5.1", Sequence: 228`)
	require.NotContains(t, string(before), `Version: "5.0"`)

	// Each release line can advance independently while RCs are in flight.
	git("checkout", "release-4.10")
	write(221, "4.10.10-rc.1")
	git("checkout", "release-4.11")
	for sequence, tag := range []string{
		"4.11.1-rc.0",
		"4.11.1-rc.1",
		"4.11.1-rc.2",
		"4.11.1-rc.3",
		"4.11.1-rc.4",
	} {
		write(226+sequence, tag)
	}
	git("checkout", "-qb", "release-5.0", "refs/tags/4.11.0")
	write(227, "5.0.0-rc.2")
	git("checkout", "master")

	after, err := generate(dir, "5.1.x")
	require.NoError(t, err)
	require.Equal(t, before, after, "RC tags on multiple release branches must not change GA metadata")
}

func TestNightlyVersionDiscovery(t *testing.T) {
	dir, git, write := repository(t)
	versionMakefile(t, dir)
	t.Chdir(dir)
	t.Setenv("BUILD_TAG", "")
	write(220, "4.10.0")
	write(225, "4.11.0")
	write(226, "5.0.0")
	write(227, "")
	git("tag", "-a", "5.1.x", "-m", "development")
	write(228, "")
	git("tag", "-a", "5.1.x-nightly-20260930", "-m", "nightly")
	tag, err := command(dir, "make", "--quiet", "tag")
	require.NoError(t, err)
	require.Contains(t, string(tag), "5.1.x-1-g")
	require.NotContains(t, string(tag), "nightly")
	require.NoError(t, run(""))
	before, err := os.ReadFile("pkg/migrations/release_versions.go")
	require.NoError(t, err)
	t.Setenv("BUILD_TAG", "5.1.x-nightly-20260930")
	require.NoError(t, run(""))
	after, err := os.ReadFile("pkg/migrations/release_versions.go")
	require.NoError(t, err)
	require.Equal(t, before, after, "explicit nightly BUILD_TAG must select the same stream")
	require.NoError(t, os.WriteFile(sequencePath, []byte("package internal\nvar CurrentDBVersionSeqNum = 229\n"), 0644))
	t.Setenv("BUILD_TAG", "")
	tag, err = command(dir, "make", "--quiet", "tag")
	require.NoError(t, err)
	require.Contains(t, string(tag), "-dirty")
	require.NoError(t, run(""))
	after, err = os.ReadFile("pkg/migrations/release_versions.go")
	require.NoError(t, err)
	require.Contains(t, string(after), `Version: "5.1", Sequence: 229`)
}

func TestPatchVersionSuffixes(t *testing.T) {
	dir, _, write := repository(t)
	write(220, "4.10.0")
	write(225, "4.11.0")
	write(226, "5.0.0")
	write(227, "5.1.0")
	write(228, "5.1.1")
	baseline, err := generate(dir, "5.1.0")
	require.NoError(t, err)
	for name, target := range map[string]string{
		"patch RC":               "5.1.2-rc.1",
		"patch nightly":          "5.1.1-nightly-20260930",
		"RC nightly":             "5.1.2-rc.1-nightly-20260930",
		"patch git describe":     "5.1.1-12-gabcdef-dirty",
		"release branch nightly": "5.1.x-nightly-20260930",
	} {
		t.Run(name, func(t *testing.T) {
			data, err := generate(dir, target)
			require.NoError(t, err)
			require.Equal(t, baseline, data)
		})
	}
}

func TestInvalidHistoricalSequence(t *testing.T) {
	for name, sequence := range map[string]int{"invalid": 0, "decreasing": 230} {
		t.Run(name, func(t *testing.T) {
			dir, _, write := repository(t)
			write(sequence, "4.10.0")
			write(225, "4.11.0")
			write(226, "5.0.0")
			write(227, "5.1.x")
			_, err := generate(dir, "5.1.x")
			require.Error(t, err)
		})
	}
}

func TestPhantomHistoricalStream(t *testing.T) {
	dir, _, write := repository(t)
	write(220, "4.10.0")
	write(225, "4.12.0")
	write(227, "5.1.x")
	_, err := generate(dir, "5.1.x")
	require.ErrorContains(t, err, "4.12")
}

func TestCommandIgnoresSuccessfulStderr(t *testing.T) {
	output, err := command(t.TempDir(), "sh", "-c", "printf '5.1.x'; printf 'warning' >&2")
	require.NoError(t, err)
	require.Equal(t, "5.1.x", string(output))
}

func TestMajorBoundaryAndReleaseCandidate(t *testing.T) {
	dir, _, write := repository(t)
	write(211, "4.8.0")
	write(213, "4.9.0")
	write(220, "4.10.0")
	write(225, "4.11.0")
	write(227, "5.0.0-rc.1")
	data, err := generate(dir, "5.0.0-rc.1")
	require.NoError(t, err)
	require.Contains(t, string(data), `Version: "5.0", Sequence: 227`)
	write(228, "5.0.0-rc.2")
	data, err = generate(dir, "5.0.0-rc.2")
	require.NoError(t, err)
	require.Contains(t, string(data), `Version: "5.0", Sequence: 228`)
}

func TestShallowCheckout(t *testing.T) {
	dir, _, write := repository(t)
	write(220, "4.10.0")
	write(227, "5.1.x")
	clone := filepath.Join(t.TempDir(), "clone")
	_, err := command("", "git", "clone", "--depth=1", "file://"+dir, clone)
	require.NoError(t, err)
	_, err = generate(clone, "5.1.x")
	require.ErrorContains(t, err, "full checkout")
}

func TestParseSequence(t *testing.T) {
	for name, tc := range map[string]struct {
		source string
		want   int
	}{
		"typed":      {"package internal\nvar CurrentDBVersionSeqNum int = 205", 205},
		"untyped":    {"package internal\nvar CurrentDBVersionSeqNum = 227", 227},
		"missing":    {"package internal", 0},
		"malformed":  {"not Go", 0},
		"negative":   {"package internal\nvar CurrentDBVersionSeqNum = -1", 0},
		"expression": {"package internal\nvar CurrentDBVersionSeqNum = 200 + 1", 0},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := parseSequence([]byte(tc.source))
			if tc.want == 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}
