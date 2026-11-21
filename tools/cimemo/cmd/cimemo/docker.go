// Docker image receipts: `cimemo docker build -t name:tag .` memoized
// at the IMAGE level. The receipt records the built image ID; a hit
// re-tags it instead of rebuilding. This is the CA branch's proven
// fast path (receipt → re-tag), generalized.
//
// Scope: local-machine receipts (image must exist locally to re-tag).
// Cross-machine: `docker save` the image into the blob store on
// record, `docker load` on restore — the CA work's docker layer cache
// (gocache-serve docker-push/pull) is the ported form of exactly that.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// dockerBuildTags extracts -t/--tag values from a docker/buildx build
// command line.
func dockerBuildTags(args []string) []string {
	var tags []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-t" || a == "--tag" {
			if i+1 < len(args) {
				tags = append(tags, args[i+1])
				i++
			}
			continue
		}
		if strings.HasPrefix(a, "-t=") || strings.HasPrefix(a, "--tag=") {
			tags = append(tags, strings.SplitN(a, "=", 2)[1])
		}
	}
	return tags
}

// isDockerBuild reports whether args is a docker (or buildx) build.
func isDockerBuild(args []string) bool {
	if len(args) < 3 {
		return false
	}
	switch args[0] {
	case "docker":
		return args[1] == "build" || (args[1] == "buildx" && args[2] == "build")
	case "buildx":
		return args[1] == "build"
	}
	return false
}

// dockerImageID returns the ID of the most recently relevant image for
// a tag (digest if available).
func dockerImageID(tag string) (string, error) {
	out, err := exec.Command("docker", "inspect", "--format", "{{.Id}}", tag).Output()
	if err != nil {
		return "", fmt.Errorf("docker inspect %s: %w", tag, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// dockerRetag re-tags a stored image ID with the given tag.
func dockerRetag(imageID, tag string) error {
	return exec.Command("docker", "tag", imageID, tag).Run()
}

// dockerContextClosure hashes the build context directory (every
// regular file, bounded). For image builds the context IS the input —
// including files that are outputs of previous commands (binaries
// COPYied into the image are inputs to THIS command). Git-tracked
// fallback is unsound here: untracked context files would be invisible.
func dockerContextClosure(wd string, args []string) []string {
	// find the context arg: last non-flag arg of the docker build cmd
	ctx := "."
	for i := len(args) - 1; i >= 2; i-- {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			continue
		}
		ctx = a
		break
	}
	var files []string
	filepath.WalkDir(filepath.Join(wd, ctx), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(wd, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		// Skip cimemo's own cache (self-referential: every run rewrites
		// manifests — permanent misses) and .git (stable but huge).
		// MUST run before the regular-file filter — dirs hit this path.
		if rel == ".cimemo" || rel == ".git" {
			return filepath.SkipDir
		}
		if !d.Type().IsRegular() {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	sort.Strings(files)
	if len(files) > 20000 {
		files = files[:20000] // bound; dockerignore handling is future work
	}
	return files
}

// commitReceiptKey derives the git-commit fast-path key: a clean tree
// at HEAD is EXACTLY determined (SHA → tree → contents), so the
// content closure is implied without hashing files or running
// `go list`. Sound only with a clean tree — callers must verify.
func commitReceiptKey(wd, keyCommand, toolchain string) (string, bool) {
	out, err := exec.Command("git", "-C", wd, "-c", "safe.directory="+wd, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", false
	}
	head := strings.TrimSpace(string(out))
	// clean tree? (uncommitted changes fall back to the content path)
	if err := exec.Command("git", "-C", wd, "-c", "safe.directory="+wd,
		"diff-index", "--quiet", "HEAD", "--").Run(); err != nil {
		return "", false
	}
	h := sha256.Sum256([]byte(head + "\x00" + keyCommand + "\x00" + toolchain))
	return hex.EncodeToString(h[:])[:16], true
}

// dockerBaseDigests resolves the digests of every FROM image in the
// Dockerfile (via docker inspect). Base tags are MUTABLE (alpine:3
// moves) — an image receipt keyed without them serves stale images
// after a base update. Digest-pinned FROMs are already exact.
func dockerBaseDigests(wd string, args []string) []string {
	// find -f/--file
	dockerfile := "Dockerfile"
	for i := 0; i < len(args); i++ {
		if args[i] == "-f" || args[i] == "--file" {
			if i+1 < len(args) {
				dockerfile = args[i+1]
			}
		}
	}
	data, err := os.ReadFile(filepath.Join(wd, dockerfile))
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && strings.EqualFold(f[0], "FROM") {
			image := f[1]
			if strings.Contains(image, "@sha256:") {
				out = append(out, image) // digest-pinned: exact already
				continue
			}
			insp, err := exec.Command("docker", "inspect", "--format", "{{index .RepoDigests 0}}", image).Output()
			if err == nil {
				out = append(out, strings.TrimSpace(string(insp)))
			} else {
				// unresolvable (not pulled): fall back to the tag string
				out = append(out, image)
			}
		}
	}
	return out
}
