package key

import (
	"os/exec"
	"strings"
)

// BuildConfig returns the toolchain + build-configuration identity for
// keying Go commands: go version plus CGO_ENABLED, GOOS, GOARCH and
// GOFLAGS. These change outputs without changing any input file
// content, so they must be part of the key (they are also folded into
// go's own ActionIDs — same soundness concern, applied at our layer).
func BuildConfig(dir string) string {
	parts := []string{GoVersion(dir)}
	for _, v := range []string{"CGO_ENABLED", "GOOS", "GOARCH", "GOFLAGS"} {
		cmd := exec.Command("go", "env", v)
		cmd.Dir = dir
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		parts = append(parts, v+"="+strings.TrimSpace(string(out)))
	}
	return strings.Join(parts, " ")
}
