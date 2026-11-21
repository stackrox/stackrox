package key

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitlinkMarker identifies submodule pseudo-inputs in read-sets
// ("path@gitlink:<sha>", produced by gitTrackedFiles — submodules are
// directories, not files, so they ride the read-set as markers).
const GitlinkMarker = "@gitlink:"

// HashInput hashes one read-set entry: a file by content, a gitlink by
// the submodule's current HEAD. Shared by key computation, provenance
// recording, and validation so gitlinks stay consistent everywhere.
func HashInput(dir, rel string) (string, bool) {
	if i := strings.Index(rel, GitlinkMarker); i > 0 {
		sub := rel[:i]
		p := filepath.Join(dir, sub)
		// safe.directory: tar-copied/container checkouts trip git's
		// ownership check (same fix as cmd's git calls).
		out, err := exec.Command("git", "-C", p, "-c", "safe.directory="+p, "rev-parse", "HEAD").Output()
		if err != nil {
			return "", false
		}
		return strings.TrimSpace(string(out)), true
	}
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), true
}
