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
