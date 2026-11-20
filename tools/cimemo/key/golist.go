package key

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// The go list template that yields every input file of every package
// in the dependency closure: Go sources, cgo sources, assembly,
// headers, test files, and embed patterns (expanded separately).
const goListFmt = `{{range .GoFiles}}{{$.Dir}}/{{.}}
{{end}}{{range .CgoFiles}}{{$.Dir}}/{{.}}
{{end}}{{range .SFiles}}{{$.Dir}}/{{.}}
{{end}}{{range .HFiles}}{{$.Dir}}/{{.}}
{{end}}{{range .TestGoFiles}}{{$.Dir}}/{{.}}
{{end}}{{range .EmbedPatterns}}{{$.Dir}}/{{.}}
{{end}}`

// GoClosure returns the project-relative set of input files for the
// given package patterns, computed with `go list -deps`. This is the
// exact dependency closure — no observation needed — which makes
// cache entries derived from it sound: the closure can be recomputed
// deterministically before the command runs.
//
// go.mod and go.sum are always included (they gate every compile), and
// the caller is expected to fold GoVersion() into the key so toolchain
// changes invalidate.
func GoClosure(dir string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	args := append([]string{"list", "-deps", "-f", goListFmt}, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("go list: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("go list: %w", err)
	}

	// Resolve the project root the same way file paths will be
	// reported: symlink-free. On macOS /var → /private/var etc.,
	// and go list reports resolved paths.
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		real = dir
	}

	seen := make(map[string]bool)
	addFile := func(abs string) {
		rel, err := filepath.Rel(real, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return // outside the project (module cache, stdlib)
		}
		seen[filepath.ToSlash(rel)] = true
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.ContainsAny(line, "*?[") {
			// Embed pattern — expand and keep project-local matches.
			matches, _ := filepath.Glob(line)
			for _, m := range matches {
				if info, err := os.Stat(m); err == nil && !info.IsDir() {
					addFile(m)
				}
			}
			continue
		}
		addFile(line)
	}

	// go.mod / go.sum gate every compile.
	for _, mod := range []string{"go.mod", "go.sum"} {
		if _, err := os.Stat(filepath.Join(dir, mod)); err == nil {
			seen[mod] = true
		}
	}

	out2 := make([]string, 0, len(seen))
	for f := range seen {
		out2 = append(out2, f)
	}
	sort.Strings(out2)
	return out2, nil
}

// GoVersion returns the toolchain identity ("go version" output).
// Folded into the key so a toolchain upgrade invalidates entries.
func GoVersion(dir string) string {
	cmd := exec.Command("go", "version")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GoPatterns extracts package patterns from a go command's arguments.
// Returns nil if the command is not a go compile-ish command.
//
// Known flags that take a value are skipped along with the value;
// everything else that is not a flag and looks like a package pattern
// ("./...", ".", "cmd/x", ...) is collected. If none are found the
// pattern is "." (go build with no args builds the cwd package).
func GoPatterns(args []string) []string {
	if len(args) == 0 || args[0] != "go" || len(args) < 2 {
		return nil
	}
	switch args[1] {
	case "build", "test", "vet", "install", "generate", "run", "fmt", "lint":
	default:
		return nil
	}

	// Flags that consume the following argument.
	valueFlags := map[string]bool{
		"-o": true, "-mod": true, "-tags": true, "-compiler": true,
		"-gcflags": true, "-ldflags": true, "-asmflags": true,
		"-coverpkg": true, "-C": true,
	}

	var patterns []string
	rest := args[2:]
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		if strings.HasPrefix(a, "-") {
			if valueFlags[a] && i+1 < len(rest) {
				i++ // skip the flag's value
			}
			continue
		}
		if a == "." || a == "..." || strings.HasPrefix(a, "./") ||
			strings.HasPrefix(a, "../") || strings.Contains(a, "/") || strings.HasSuffix(a, "...") {
			patterns = append(patterns, a)
		}
	}
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	return patterns
}
