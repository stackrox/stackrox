package main

// TestLinkKeyMatchesToolchain is the canary for vendored linkActionID:
// it builds a tiny program with GODEBUG=gocachehash=1, captures the go
// command's OWN dump of the link hash inputs, and verifies:
//
//   Level 1 (hash mechanics): salt + concatenated dump writes == dumped sum
//   Level 2 (vendored renderer): structured LinkKeyInput parsed from the
//     dump reproduces the exact byte sequence
//   Level 3 (end-to-end): HashToString(sum) == the binary's actual
//     actionID (part 1 of its buildID, read via go tool buildid)
//
// If the toolchain changes its key generation (new inputs, new format,
// different salt), any level fails loudly — update linkkey.go then. The
// binary buildID check still validates every cache skip in production,
// so a stale vendored key degrades to a relink, never a wrong skip.

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLinkKeyMatchesToolchain(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a program; skip in -short")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not in PATH")
	}

	// Cross-compile coverage: the test never executes the built binary
	// (only reads its buildID note), so any GOOS/GOARCH works from any
	// host — no docker needed.
	for _, plat := range []struct{ goos, goarch string }{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
	} {
		t.Run(plat.goos+"/"+plat.goarch, func(t *testing.T) {
			testLinkKeyPlatform(t, goBin, plat.goos, plat.goarch)
		})
	}
}

func testLinkKeyPlatform(t *testing.T, goBin, goos, goarch string) {

	dir := t.TempDir()
	write := func(rel, content string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/linkkeytest\n\ngo 1.27\n")
	write("main.go", "package main\n\nimport (\n\t\"fmt\"\n\t\"example.com/linkkeytest/lib\"\n)\n\nfunc main() { fmt.Println(lib.V()) }\n")
	write("lib/lib.go", "package lib\n\nfunc V() string { return \"v\" }\n")

	cmd := exec.Command(goBin, "build", "-o", "out", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GODEBUG=gocachehash=1", "GOFLAGS=-buildvcs=false", "GOOS="+goos, "GOARCH="+goarch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// Parse the dump: HASH[name]: "quoted-write" lines and final sums.
	// The link hash is computed MORE THAN ONCE per build (action planning,
	// then execution with final dep buildIDs) — segment writes into rounds
	// at each sum line; the final round is the authoritative linkActionID.
	const linkName = "link example.com/linkkeytest"
	var rounds [][]string
	var sums []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "HASH[") {
			continue
		}
		rest := line[len("HASH["):]
		bracket := strings.Index(rest, "]")
		if bracket < 0 {
			continue
		}
		name := rest[:bracket]
		if name != linkName {
			continue
		}
		val := strings.TrimSpace(rest[bracket+1:])
		val = strings.TrimPrefix(val, ":")
		val = strings.TrimSpace(val)
		if s, err := strconv.Unquote(val); err == nil {
			if len(rounds) == 0 {
				rounds = [][]string{{}}
			}
			rounds[len(rounds)-1] = append(rounds[len(rounds)-1], s)
		} else if sum := strings.TrimSpace(val); len(sum) == 64 && !strings.Contains(sum, "\"") {
			sums = append(sums, sum)
			rounds = append(rounds, []string{}) // start next round
		}
	}
	// Drop the trailing empty round; the final real round is authoritative.
	if len(rounds) > 0 && len(rounds[len(rounds)-1]) == 0 {
		rounds = rounds[:len(rounds)-1]
	}
	if len(rounds) == 0 || len(sums) == 0 {
		t.Fatalf("could not parse link hash dump\noutput:\n%s", out)
	}
	writes := rounds[len(rounds)-1]
	sumHex := sums[len(sums)-1]
	if len(writes) < 10 {
		t.Fatalf("too few writes in final round (%d)\noutput:\n%s", len(writes), out)
	}
	// First write is the salt (the go version); the rest is the sequence.
	if len(writes[0]) == 0 || !strings.HasPrefix(writes[0], "go1.") {
		t.Fatalf("first write is not the version salt: %q", writes[0])
	}

	// Level 1: salt + writes -> dumped sum.
	h := sha256.New()
	for _, w := range writes {
		h.Write([]byte(w))
	}
	got := fmt.Sprintf("%x", h.Sum(nil))
	if got != sumHex {
		for i, w := range writes {
			t.Logf("write[%d] = %q", i, w)
		}
		t.Fatalf("level 1 mismatch:\n computed %s\n dumped   %s\n (writes=%d rounds=%d)", got, sumHex, len(writes), len(rounds))
	}

	// Level 2: structured input parsed from the dump reproduces the bytes.
	in := LinkKeyInput{GoVersion: writes[0]}
	var deps []LinkKeyDep
	for _, w := range writes[1:] {
		switch {
		case w == "link\n":
		case strings.HasPrefix(w, "buildmode "):
			fmt.Sscanf(w, "buildmode %s goos %s goarch %s", &in.Buildmode, &in.Goos, &in.Goarch)
		case strings.HasPrefix(w, "import "):
			in.ImportPath = unquotePrefix(t, strings.TrimPrefix(w, "import "))
		case strings.HasPrefix(w, "omitdebug "):
			var od, st, lc bool
			var pfx string
			fmt.Sscanf(w, "omitdebug %v standard %v local %v prefix %q", &od, &st, &lc, &pfx)
			in.OmitDebug, in.Standard, in.Local, in.LocalPrefix = od, st, lc, pfx
		case strings.HasPrefix(w, "defaultgodebug "):
			in.DefaultGODEBUG = unquotePrefix(t, strings.TrimPrefix(w, "defaultgodebug "))
		case w == "trimpath\n":
			in.Trimpath = true
		case strings.HasPrefix(w, "link "):
			// Format: "link <toolID...> <ldflags %q> <ldBuildmode>\n" where
			// toolID is multi-word ("link version go1.27.1"). Parse from
			// the right: last token = ldBuildmode, second-to-last quoted
			// = ldflags, the rest = toolID.
			body := strings.TrimSuffix(strings.TrimPrefix(w, "link "), "\n")
			fields := strings.Split(body, " ")
			if len(fields) < 3 {
				t.Fatalf("malformed link config line: %q", w)
			}
			in.LdBuildmode = fields[len(fields)-1]
			lf := fields[len(fields)-2]
			if lf == "[]" {
				in.ForcedLdflags = nil // %q of an empty []string
				in.ToolID = strings.Join(fields[:len(fields)-2], " ")
			} else if u, err := strconv.Unquote(lf); err == nil {
				var parts []string
				f := strings.Trim(u, "[]")
				if f != "" {
					for _, p := range strings.Fields(f) {
						if pu, err := strconv.Unquote(p); err == nil {
							parts = append(parts, pu)
						}
					}
				}
				in.ForcedLdflags = parts
				in.ToolID = strings.Join(fields[:len(fields)-2], " ")
			} else {
				t.Fatalf("malformed ldflags in link line: %q", w)
			}
		case strings.HasPrefix(w, "linkflags "):
			f := strings.Trim(strings.TrimPrefix(w, "linkflags "), "[]")
			var parts []string
			for _, p := range strings.Fields(f) {
				if u, err := strconv.Unquote(p); err == nil {
					parts = append(parts, u)
				}
			}
			in.PackageLdflags = parts
		case strings.HasPrefix(w, "GOEXPERIMENT="):
			in.GOEXPERIMENT = strings.Trim(strings.TrimPrefix(w, "GOEXPERIMENT="), "\"")
		case strings.HasPrefix(w, "GOROOT="):
			in.Goroot = strings.TrimSuffix(strings.TrimPrefix(w, "GOROOT="), "\n")
		case strings.HasPrefix(w, "GO_EXTLINK_ENABLED="):
			in.GoExtlinkEnabled = strings.TrimSuffix(strings.TrimPrefix(w, "GO_EXTLINK_ENABLED="), "\n")
		case strings.HasPrefix(w, "packagefile "):
			kv := strings.SplitN(strings.TrimPrefix(w, "packagefile "), "=", 2)
			if len(kv) == 2 {
				deps = append(deps, LinkKeyDep{ImportPath: strings.TrimSuffix(kv[0], "\n"), ContentID: strings.TrimSuffix(kv[1], "\n")})
			}
		case strings.HasPrefix(w, "packagemain "):
			in.MainBuildID = strings.TrimSpace(strings.TrimPrefix(w, "packagemain "))
			// Emitted inside the dep loop, right after the main package's
			// packagefile — mark that dep.
			if len(deps) > 0 {
				deps[len(deps)-1].IsMain = true
			}
		default:
			// arch env (GOAMD64=v1 / GOARM64=v8.0 class) and anything new:
			// if go adds an input line we don't know, fail loudly — that's
			// the canary doing its job.
			w = strings.TrimSuffix(w, "\n")
			if strings.Contains(w, "=") {
				kv := strings.SplitN(w, "=", 2)
				in.ArchEnvKey, in.ArchEnvVal = kv[0], kv[1]
			} else {
				t.Fatalf("unrecognized hash input line (toolchain changed?): %q", w)
			}
		}
	}
	in.Deps = deps
	sum2 := in.Hash()
	if fmt.Sprintf("%x", sum2) != sumHex {
		rendered := in.RenderBytes()
		dumped := []byte(strings.Join(writes[1:], ""))
		i := 0
		for i < len(rendered) && i < len(dumped) && rendered[i] == dumped[i] {
			i++
		}
		lo := i - 40
		if lo < 0 {
			lo = 0
		}
		hiR, hiD := i+40, i+40
		if hiR > len(rendered) {
			hiR = len(rendered)
		}
		if hiD > len(dumped) {
			hiD = len(dumped)
		}
		t.Fatalf("level 2 mismatch at byte %d:\n rendered ...%q\n dumped   ...%q", i, rendered[lo:hiR], dumped[lo:hiD])
	}

	// Level 3: the binary's actual actionID (buildID part 1).
	bid, err := exec.Command(goBin, "tool", "buildid", filepath.Join(dir, "out")).Output()
	if err != nil {
		t.Fatalf("go tool buildid: %v", err)
	}
	parts := strings.Split(strings.TrimSpace(string(bid)), "/")
	if len(parts) < 4 {
		t.Fatalf("unexpected buildID %q", string(bid))
	}
	if act := LinkKeyActionID(sum2); act != parts[0] {
		t.Fatalf("level 3 mismatch: actionID %s != binary buildID part %s", act, parts[0])
	}
}

func unquotePrefix(t *testing.T, s string) string {
	t.Helper()
	u, err := strconv.Unquote(s)
	if err != nil {
		// tolerate trailing newline inside the quoted string
		if i := strings.LastIndex(s, "\""); i > 0 {
			if u2, err2 := strconv.Unquote(s[:i+1]); err2 == nil {
				return u2
			}
		}
		t.Fatalf("unquote %q: %v", s, err)
	}
	return u
}
