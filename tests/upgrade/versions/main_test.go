package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stretchr/testify/require"
)

func TestRollbackPlan(t *testing.T) {
	for name, tc := range map[string]struct {
		target, from, floor, rejected string
		hops                          []string
	}{
		"current":         {"5.1.x-12-gabc-dirty", "4.11.3", "4.10", "4.9", nil},
		"major boundary":  {"5.2.0-rc.1", "4.11.3", "4.11", "4.10", nil},
		"cross major hop": {"5.3.x-nightly-20260930", "4.11.3", "5.0", "4.11", []string{"5.0.0"}},
		"multiple hops":   {"5.4.2", "4.11.3", "5.1", "5.0", []string{"5.0.0", "5.1.0"}},
	} {
		t.Run(name, func(t *testing.T) {
			floor := func(string) (migrations.ReleaseVersion, error) {
				return migrations.ReleaseVersion{Version: tc.floor, Sequence: 220}, nil
			}
			lookup := func(tag string) (release, error) {
				seq := 220
				if tag == tc.rejected+".0" {
					seq = 213
				}
				return release{Tag: tag, SHA: "commit-" + tag, Sequence: seq}, nil
			}
			got, err := makePlan(tc.target, tc.from, floor, lookup)
			require.NoError(t, err)
			require.Equal(t, tc.floor+".0", got.Allowed.Tag)
			require.Equal(t, tc.rejected+".0", got.Rejected.Tag)
			require.Equal(t, 220, got.MinimumSequence)
			var hops []string
			for _, r := range got.AdditionalUpgrades {
				hops = append(hops, r.Tag)
			}
			require.Equal(t, tc.hops, hops)
		})
	}
}

func TestReadRelease(t *testing.T) {
	for name, tc := range map[string]struct {
		source string
		want   int
	}{
		"untyped":    {"package internal\nconst CurrentDBVersionSeqNum = 220", 220},
		"typed":      {"package internal\nconst CurrentDBVersionSeqNum int = 220", 220},
		"missing":    {"package internal\nconst AnotherSequence = 220", 0},
		"zero":       {"package internal\nconst CurrentDBVersionSeqNum = 0", 0},
		"expression": {"package internal\nconst CurrentDBVersionSeqNum = 219 + 1", 0},
		"malformed":  {"not Go source", 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			git := func(args ...string) string {
				t.Helper()
				out, err := exec.Command("git", append([]string{"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
				require.NoError(t, err, "%s", out)
				return string(out)
			}
			git("init", "--quiet")
			path := filepath.Join("pkg", "migrations", "internal", "seq_num.go")
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte(tc.source), 0o644))
			git("add", path)
			git("commit", "--quiet", "-m", "initial release")
			git("tag", "4.10.0")
			// A patch migration and dirty working tree must not change the GA baseline.
			require.NoError(t, os.WriteFile(path, []byte("package internal\nconst CurrentDBVersionSeqNum = 999"), 0o644))
			git("add", path)
			git("commit", "--quiet", "-m", "patch migration")
			git("tag", "4.10.1")
			require.NoError(t, os.WriteFile(path, []byte("dirty"), 0o644))
			got, err := readRelease("4.10.0")
			if tc.want == 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got.Sequence)
				require.Equal(t, git("rev-parse", "4.10.0")[:40], got.SHA)
			}
			_, err = readRelease("4.11.0")
			require.ErrorContains(t, err, "missing GA tag")
		})
	}
}

func TestRollbackPlanMissingHop(t *testing.T) {
	_, err := makePlan("5.5.0", "4.11.3", func(string) (migrations.ReleaseVersion, error) {
		return migrations.ReleaseVersion{Version: "5.2", Sequence: 250}, nil
	}, func(tag string) (release, error) {
		if tag == "5.0.0" {
			return release{}, fmt.Errorf("missing hop %s", tag)
		}
		if tag == "5.1.0" {
			return release{Tag: tag, Sequence: 240}, nil
		}
		return release{Tag: tag, Sequence: 250}, nil
	})
	require.ErrorContains(t, err, "missing hop 5.0.0")
}

func TestRollbackPlanFailures(t *testing.T) {
	for name, tc := range map[string]struct {
		missing           string
		allowed, rejected int
	}{
		"equal sequences":      {allowed: 220, rejected: 220},
		"higher old sequence":  {allowed: 220, rejected: 221},
		"baseline mismatch":    {allowed: 221, rejected: 213},
		"missing floor tag":    {missing: "4.10.0"},
		"missing rejected tag": {allowed: 220, missing: "4.9.0"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := makePlan("5.1.x", "4.11.3", migrations.MinimumSupportedForVersion, func(tag string) (release, error) {
				if tag == tc.missing {
					return release{}, fmt.Errorf("missing tag %s", tag)
				}
				seq := tc.allowed
				if tag == "4.9.0" {
					seq = tc.rejected
				}
				return release{Tag: tag, Sequence: seq}, nil
			})
			require.Error(t, err)
		})
	}
	_, err := makePlan("bad", "4.11.3", migrations.MinimumSupportedForVersion, nil)
	require.Error(t, err)
}
