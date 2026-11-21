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
	"os/exec"
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
