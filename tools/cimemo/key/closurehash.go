package key

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
)

// ClosureHash hashes ONLY the input closure (no command, no toolchain,
// no VCS) — the "same files, allegedly same build" identity used by
// the nondeterminism detector: two manifests with the same closure
// hash but different keys or different outputs mean something outside
// the closure changed (or the build is nondeterministic).
func ClosureHash(dir string, files []string) string {
	h := sha256.New()
	sorted := append([]string{}, files...)
	sort.Strings(sorted)
	for _, f := range sorted {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			continue
		}
		sum := sha256.Sum256(data)
		fmt := sum[:]
		h.Write([]byte(f))
		h.Write(fmt)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
