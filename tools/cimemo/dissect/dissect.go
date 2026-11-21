// Package dissect explains why two binaries differ despite (allegedly)
// identical inputs — a diffoscope-lite for cache-divergence triage.
//
// Layers:
//  1. Section-level hashing (stdlib ELF/Mach-O parsers) — which parts
//     of the binary changed, not just that it did.
//  2. Toolchain knowledge — names likely causes from which sections
//     differ (see likelyCauses).
//  3. Content heuristics — date-like strings inside differing
//     sections (embedded timestamps, vcs.time, -X date stamps).
//
// Toolchain knowledge (go is implemented; others documented for their
// case studies):
//
//	go:     vcs.* stamps, -ldflags -X, go:embed, build ID, DWARF paths
//	rust:   TODO(case study) — vergen/build.rs embedded timestamps and
//	        commit SHAs in .rodata; DWARF comp_dir paths (fix:
//	        --remap-path-prefix); .comment compiler version; GNU
//	        build-id. Detection identical to go's via section diff +
//	        date-string scan.
//	c/c++:  TODO(case study) — __DATE__/__TIME__ macros into .rodata;
//	        DWARF .debug_str comp_dir (-ffile-prefix-map);
//	        .comment gcc version; `ar` archives embed MEMBER MTIMES
//	        (the same mtime-instability class as go's moduleIndex —
//	        use thin archives or deterministic ar mode).
package dissect

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"debug/elf"
	"debug/macho"
	"github.com/davdhacs/cimemo/buildinfo"
)

// SectionDiff is one differing section.
type SectionDiff struct {
	Name  string
	SizeA int64
	SizeB int64
	HashA string
	HashB string
}

// Report explains a binary divergence.
type Report struct {
	Format       string // "elf" | "macho" | "raw"
	SameSize     bool
	Sections     []SectionDiff
	BuildInfoA   *buildinfo.Info
	BuildInfoB   *buildinfo.Info
	BuildIDSame  bool
	DateStrings  []string // date-like strings inside differing sections
	LikelyCauses []string

	// ContentEquivalent: byte-different but semantically identical
	// (ar member naming). Divergence detectors must not fire on it.
	ContentEquivalent bool
}

var dateRe = regexp.MustCompile(`20\d{2}-\d{2}-\d{2}[T ]\d{2}:\d{2}(:\d{2})?`)

// Diff compares two binaries and explains the difference.
func Diff(pathA, pathB string) (*Report, error) {
	r := &Report{}
	dataA, err := os.ReadFile(pathA)
	if err != nil {
		return nil, err
	}
	dataB, err := os.ReadFile(pathB)
	if err != nil {
		return nil, err
	}
	r.SameSize = len(dataA) == len(dataB)

	// Layer 2: buildinfo (go binaries — nil for others, harmless).
	r.BuildInfoA, _ = buildinfo.Inspect(pathA)
	r.BuildInfoB, _ = buildinfo.Inspect(pathB)
	if r.BuildInfoA != nil && r.BuildInfoB != nil {
		if ok, m := buildinfo.Verify(r.BuildInfoA, r.BuildInfoB); !ok {
			r.LikelyCauses = append(r.LikelyCauses, "buildinfo mismatch: "+m)
		}
	}

	// Layer 0: ar archives (rust rlib, static libs). Members carry
	// per-build random names (rustc codegen-unit temp suffixes) while
	// content stays identical — raw byte comparison false-positives.
	if aA, okA := arMembers(dataA); okA {
		if aB, okB := arMembers(dataB); okB {
			r.Format = "ar"
			// Name-listing members (__.SYMDEF indexes, *-link linker
			// directives) embed the random member names — their bytes
			// differ even when real content matches. Compare only real
			// members.
			realA, realB := realMembers(aA), realMembers(aB)
			if multisetEqual(realA, realB) {
				r.ContentEquivalent = true
				r.LikelyCauses = append(r.LikelyCauses,
					fmt.Sprintf("archive member CONTENT identical (%d real members) — only member names and name-listing members differ (rust codegen-unit random suffixes); outputs are semantically identical", len(realA)))
				return r, nil
			}
			r.LikelyCauses = append(r.LikelyCauses,
				fmt.Sprintf("ar member contents differ: %d vs %d real members, %d shared", len(realA), len(realB), sharedCount(realA, realB)))
		}
	}

	// Layer 1: section diff.
	sectionsA, fmtA := readSections(pathA)
	sectionsB, fmtB := readSections(pathB)
	if fmtA != "" && fmtA == fmtB {
		r.Format = fmtA
		for name, hA := range sectionsA {
			hB, ok := sectionsB[name]
			if !ok {
				r.Sections = append(r.Sections, SectionDiff{Name: name, HashA: hA})
				continue
			}
			if hA != hB {
				r.Sections = append(r.Sections, SectionDiff{Name: name, HashA: hA, HashB: hB})
			}
		}
		for name, hB := range sectionsB {
			if _, ok := sectionsA[name]; !ok {
				r.Sections = append(r.Sections, SectionDiff{Name: name, HashB: hB})
			}
		}
		// Layer 3: date-like strings in differing sections.
		for _, d := range r.Sections {
			if data := sectionData(pathA, d.Name); data != nil {
				for _, m := range dateRe.FindAll(data, 8) {
					r.DateStrings = append(r.DateStrings, string(m))
				}
			}
		}
	} else {
		r.Format = "raw"
		// Whole-file fallback: first differing offset.
		if !bytes.Equal(dataA, dataB) {
			i := 0
			for i < len(dataA) && i < len(dataB) && dataA[i] == dataB[i] {
				i++
			}
			r.LikelyCauses = append(r.LikelyCauses,
				fmt.Sprintf("first difference at offset %d (sizes %d vs %d)", i, len(dataA), len(dataB)))
		}
	}

	r.diagnose()
	return r, nil
}

// readSections returns section name → content hash.
func readSections(path string) (map[string]string, string) {
	if f, err := elf.Open(path); err == nil {
		defer f.Close()
		return hashElfSections(f.Sections), "elf"
	}
	if f, err := macho.Open(path); err == nil {
		defer f.Close()
		return hashMachoSections(f.Sections), "macho"
	}
	return nil, ""
}

func hashElfSections(secs []*elf.Section) map[string]string {
	out := map[string]string{}
	for _, s := range secs {
		data, err := s.Data()
		if err != nil || len(data) == 0 {
			continue
		}
		sum := sha256.Sum256(data)
		out[s.Name] = hex.EncodeToString(sum[:16])
	}
	return out
}

func hashMachoSections(secs []*macho.Section) map[string]string {
	out := map[string]string{}
	for _, s := range secs {
		data, err := s.Data()
		if err != nil || len(data) == 0 {
			continue
		}
		sum := sha256.Sum256(data)
		out[s.Name] = hex.EncodeToString(sum[:16])
	}
	return out
}

func sectionData(path, name string) []byte {
	if f, err := elf.Open(path); err == nil {
		defer f.Close()
		if s := f.Section(name); s != nil {
			data, _ := s.Data()
			return data
		}
	}
	if f, err := macho.Open(path); err == nil {
		defer f.Close()
		if s := f.Section(name); s != nil {
			data, _ := s.Data()
			return data
		}
	}
	return nil
}

// diagnose maps differing sections to likely causes (the knowledge layer).
func (r *Report) diagnose() {
	has := func(prefix string) bool {
		for _, d := range r.Sections {
			if len(d.Name) >= len(prefix) && d.Name[:len(prefix)] == prefix {
				return true
			}
		}
		return false
	}
	for _, d := range r.Sections {
		switch {
		case d.Name == ".comment":
			r.LikelyCauses = append(r.LikelyCauses,
				"compiler version changed (.comment) — toolchain not fully pinned?")
		case d.Name == ".note.go.buildid" || d.Name == "note_gnu_build_id":
			r.BuildIDSame = false
		}
	}
	if has(".debug") {
		r.LikelyCauses = append(r.LikelyCauses,
			"debug info differs (.debug_*) — source paths or line tables; fix with -trimpath / -ffile-prefix-map / --remap-path-prefix")
	}
	if has(".rodata") && len(r.DateStrings) > 0 {
		r.LikelyCauses = append(r.LikelyCauses,
			fmt.Sprintf(".rodata differs and contains date-like strings (%v) — embedded timestamps: __DATE__/__TIME__, vcs.time, or -X date vars", r.DateStrings[:min(3, len(r.DateStrings))]))
	}
	if r.BuildInfoA != nil && r.BuildInfoB != nil &&
		r.BuildInfoA.VCSRevision != "" && r.BuildInfoA.VCSRevision != r.BuildInfoB.VCSRevision {
		r.LikelyCauses = append(r.LikelyCauses,
			"different commits with identical trees (vcs.revision differs) — cherry-pick/rebase case")
	}
	if len(r.LikelyCauses) == 0 && len(r.Sections) > 0 {
		r.LikelyCauses = append(r.LikelyCauses,
			"differing sections: "+fmt.Sprint(sectionNames(r.Sections)))
	}
}

func sectionNames(diffs []SectionDiff) []string {
	var out []string
	for _, d := range diffs {
		out = append(out, d.Name)
	}
	return out
}

// String renders the report for logs.
func (r *Report) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "format=%s sameSize=%v\n", r.Format, r.SameSize)
	if len(r.Sections) > 0 {
		fmt.Fprintf(&b, "differing sections (%d): %v\n", len(r.Sections), sectionNames(r.Sections))
	}
	if len(r.DateStrings) > 0 {
		fmt.Fprintf(&b, "date-like strings: %v\n", r.DateStrings)
	}
	for _, c := range r.LikelyCauses {
		fmt.Fprintf(&b, "likely cause: %s\n", c)
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// arMembers parses a Unix ar archive into a sorted multiset of member
// body hashes. ok=false when not an ar archive or parse fails.
type arMember struct {
	name string
	hash string
}

// realMembers filters out name-listing members whose bytes are derived
// from the (random) member names.
func realMembers(ms []arMember) []string {
	var out []string
	for _, m := range ms {
		base := m.name
		if i := strings.Index(base, "."); i > 0 {
			// strip random middle suffixes: keep only the crate+hash
			// head and the extension tail for classification
		}
		if strings.HasPrefix(base, "__.SYMDEF") || strings.HasSuffix(base, "-link") {
			continue
		}
		out = append(out, m.hash)
	}
	sort.Strings(out)
	return out
}

func arMembers(data []byte) ([]arMember, bool) {
	if len(data) < 8 || string(data[:8]) != "!<arch>\n" {
		return nil, false
	}
	var out []arMember
	off := 8
	for off+60 <= len(data) {
		hdr := data[off : off+60]
		if string(hdr[58:60]) != "`\n" {
			return nil, false
		}
		size, err := strconv.Atoi(strings.TrimSpace(string(hdr[48:58])))
		if err != nil || off+60+size > len(data) {
			return nil, false
		}
		body := data[off+60 : off+60+size]
		name := strings.TrimSpace(string(hdr[:16]))
		// BSD long names: "#1/<len>" — the member name occupies the
		// first <len> bytes of the body; hash only the content after it
		// (macOS ar writes rust rlibs this way, which is exactly where
		// the random member names live).
		if strings.HasPrefix(name, "#1/") {
			if n, err := strconv.Atoi(name[3:]); err == nil && n <= len(body) {
				name = strings.TrimSpace(strings.TrimRight(string(body[:n]), "\x00"))
				body = body[n:]
			}
		}
		sum := sha256.Sum256(body)
		out = append(out, arMember{name: name, hash: hex.EncodeToString(sum[:16])})
		off += 60 + size
		if size%2 == 1 {
			off++
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].hash < out[j].hash })
	return out, true
}

func multisetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sharedCount(a, b []string) int {
	set := map[string]int{}
	for _, x := range a {
		set[x]++
	}
	n := 0
	for _, x := range b {
		if set[x] > 0 {
			set[x]--
			n++
		}
	}
	return n
}
