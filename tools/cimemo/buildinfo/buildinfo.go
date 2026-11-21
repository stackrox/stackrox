// Package buildinfo inspects go binaries' embedded build metadata
// (go version -m) and turns it into memoization-safety findings.
//
// Why: cimemo keys on file content + command + build config. Two
// binary inputs are invisible to that key:
//
//   - VCS stamping (vcs.revision/time/modified): embedded per commit.
//     A cherry-pick/rebase/empty commit yields an identical tree at a
//     different SHA — a content-keyed receipt would serve the old
//     stamp. Mitigated by key-folding git HEAD (see KeyFoldVCS).
//   - -ldflags -X values: those in argv or GOFLAGS are already keyed;
//     anything else is flagged for human review.
//
// Post-restore verification: manifest.BuildInfo lets a receipt hit
// re-inspect the restored binary and confirm it matches what was
// stored — a semantic identity check on top of the blob hash.
package buildinfo

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Info is the memoization-relevant subset of a binary's buildinfo.
type Info struct {
	VCSRevision string            `json:"vcs_revision,omitempty"`
	VCSTime     string            `json:"vcs_time,omitempty"`
	VCSModified string            `json:"vcs_modified,omitempty"`
	Ldflags     string            `json:"ldflags,omitempty"`
	GoVersion   string            `json:"go_version,omitempty"`
	Settings    map[string]string `json:"settings,omitempty"`
}

// Finding is one audit result.
type Finding struct {
	Kind    string // "vcs-stamp" | "ldflags-external" | "verify-mismatch"
	Detail  string
	Warning string // human-readable
}

// Inspect parses a binary's embedded buildinfo. Returns nil when the
// file is not a go binary (not an error — outputs can be anything).
func Inspect(binPath string) (*Info, error) {
	cmd := exec.Command("go", "version", "-m", "-json", binPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil // not a go binary, or unreadable — not our problem
	}
	var m struct {
		GoVersion string `json:"GoVersion"`
		Settings  []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"Settings"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		return nil, nil
	}
	info := &Info{GoVersion: m.GoVersion, Settings: map[string]string{}}
	for _, s := range m.Settings {
		info.Settings[s.Key] = s.Value
		switch s.Key {
		case "vcs.revision":
			info.VCSRevision = s.Value
		case "vcs.time":
			info.VCSTime = s.Value
		case "vcs.modified":
			info.VCSModified = s.Value
		case "-ldflags":
			info.Ldflags = s.Value
		}
	}
	return info, nil
}

// Audit checks a build's info against the command that produced it.
// command is the full argv cimemo ran (ldflags in argv are keyed).
func Audit(info *Info, command string) []Finding {
	if info == nil {
		return nil
	}
	var out []Finding
	if info.VCSRevision != "" {
		out = append(out, Finding{
			Kind:   "vcs-stamp",
			Detail: info.VCSRevision,
			Warning: "output embeds VCS stamps; cimemo keys on file content — " +
				"identical trees at different commits (cherry-pick/rebase) would serve the old stamp. " +
				"Key-folding git HEAD (default) makes this sound; set -buildvcs=false to avoid entirely.",
		})
	}
	if info.Ldflags != "" && !strings.Contains(command, "-ldflags") {
		out = append(out, Finding{
			Kind:   "ldflags-external",
			Detail: info.Ldflags,
			Warning: "binary was linked with -ldflags not present in the memoized command " +
				"(sourced from GOFLAGS or a wrapper) — verify the values are captured by the key, " +
				"or pin them in .cimemo.json",
		})
	}
	return out
}

// Verify compares a restored binary's buildinfo against the manifest's
// record — the post-restore semantic identity check.
func Verify(recorded, restored *Info) (ok bool, mismatch string) {
	if recorded == nil || restored == nil {
		return true, "" // nothing recorded → nothing to verify
	}
	if recorded.VCSRevision != restored.VCSRevision {
		return false, fmt.Sprintf("vcs.revision %s != %s", recorded.VCSRevision, restored.VCSRevision)
	}
	if recorded.Ldflags != restored.Ldflags {
		return false, fmt.Sprintf("ldflags %q != %q", recorded.Ldflags, restored.Ldflags)
	}
	return true, ""
}
