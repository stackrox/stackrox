// Package key computes cache keys for memoized commands.
//
// A key identifies the complete set of inputs that determine a
// command's output. If any input changes, the key changes; if none
// change, the output is identical by determinism.
package key

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Compute hashes the state of a directory tree and returns a
// 16-hex-char key. Excludes hidden files, .git, and any patterns
// in exclude.
func Compute(dir string, exclude []string) (string, error) {
	h := sha256.New()

	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		base := filepath.Base(rel)

		// Always exclude hidden files and VCS dirs
		if strings.HasPrefix(base, ".") {
			return nil
		}
		parts := strings.Split(rel, "/")
		for _, part := range parts[:len(parts)-1] {
			if strings.HasPrefix(part, ".") {
				return filepath.SkipDir
			}
		}

		// User-specified exclusions
		for _, ex := range exclude {
			if matched, _ := filepath.Match(ex, rel); matched {
				return nil
			}
			if matched, _ := filepath.Match(ex, base); matched {
				return nil
			}
		}

		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)

	// Hash each file: path + content hash
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			continue // unreadable files are skipped
		}
		sum := sha256.Sum256(data)
		contentHash := hex.EncodeToString(sum[:])
		fmt.Fprintf(h, "%s:%s\n", rel, contentHash)
	}

	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

// FromFiles computes a key from a specific list of files + the command.
// This is used when the caller knows exactly which files are inputs
// (e.g., git-tracked source files, excluding build outputs).
func FromFiles(dir string, files []string, cmd string) string {
	h := sha256.New()
	sort.Strings(files)
	for _, f := range files {
		contentHash, ok := HashInput(dir, f)
		if !ok {
			continue
		}
		fmt.Fprintf(h, "%s:%s\n", f, contentHash)
	}
	fmt.Fprintf(h, "cmd:%s", cmd)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Command computes a key that includes the tree state AND the command.
// This ensures different commands over the same tree get different keys.
func Command(dir, cmd string, exclude []string) (string, error) {
	treeKey, err := Compute(dir, exclude)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s:%s", treeKey, cmd)
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}
