package main

// diff.go computes the changed-file list of a pull request with git. Why
// the binary owns the diff instead of taking a file list from the caller:
//
//   - The pull-request files API truncates silently above 3000 files. A
//     truncated list can look like complete skip coverage and suppress
//     tests that must run.
//   - Rename detection would hide a moved file: --no-renames counts both
//     sides of a rename, so moving a backend fixture into docs cannot
//     turn the removal into a docs-only change.
//   - A failed diff must become run-everything inside this process:
//     ghaDecision turns any error from here into a run-everything
//     fallback, so fail-closed never depends on a caller honoring it.

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func changedFiles(ctx context.Context, root, base, head string) ([]string, error) {
	if base == "" || head == "" || strings.HasPrefix(base, "-") || strings.HasPrefix(head, "-") {
		return nil, errors.New("missing or invalid diff refs")
	}
	if err := fetchDiffHistory(ctx, root, base, head, "origin"); err != nil {
		return nil, err
	}
	// NUL separation keeps newline-bearing paths whole, and the
	// three-dot range measures from the merge base.
	cmd := exec.CommandContext(ctx, "git", "-C", root, "diff", "--no-renames", "--name-only", "-z", base+"..."+head, "--")
	var diagnostic strings.Builder
	cmd.Stderr = &diagnostic
	data, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrapf(err, "collect diff %s...%s: %s", base, head, strings.TrimSpace(diagnostic.String()))
	}
	if len(data) == 0 {
		return nil, nil
	}
	return strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00"), nil
}

// The initial fetch depth and the total depth cap for fetchDiffHistory. The
// depth counts ancestry, not first-parent history, so the initial depth covers
// recent pull requests; older fork points need a few doubling rounds before
// the cap fails the selection, which runs every test.

const (
	initialFetchDepth = 1000
	maxFetchDepth     = 64000
)

// fetchableRef matches the full commit SHAs that GitHub event payloads carry.
var fetchableRef = regexp.MustCompile(`^[0-9a-f]{40}$`)

// fetchDiffHistory makes the merge base of base and head resolvable without a
// full-history checkout. A repository that already contains both commits and
// their merge base fetches nothing, which also covers complete local clones.
// SHA refs are fetched from remote with deepening history until the two sides
// connect; other ref expressions pass through for the diff command to judge.
//
// The depth loop is the whole trick: a fixed depth would be one fetch, but a
// pull request that forked from the base further back than that depth would
// silently lose selection, because the merge base never materializes and the
// diff fails. Starting shallow and doubling keeps the common case one cheap
// round while old pull requests pay a few extra fetches instead of a silent
// fallback. The cap bounds the worst case; past it, selection gives up and
// the caller runs every test.
func fetchDiffHistory(ctx context.Context, root, base, head, remote string) error {
	if !fetchableRef.MatchString(base) || !fetchableRef.MatchString(head) {
		return nil
	}
	if commitExists(ctx, root, base) && commitExists(ctx, root, head) &&
		mergeBaseExists(ctx, root, base, head) {
		return nil
	}
	for depth := initialFetchDepth; ; depth *= 2 {
		if err := gitFetch(ctx, root, remote, "--no-tags", "--depth="+strconv.Itoa(depth), base, head); err != nil {
			return err
		}
		if mergeBaseExists(ctx, root, base, head) {
			return nil
		}
		if depth >= maxFetchDepth {
			return fmt.Errorf("no merge base for %s and %s within %d commits", base, head, maxFetchDepth)
		}
	}
}

func commitExists(ctx context.Context, root, ref string) bool {
	return exec.CommandContext(ctx, "git", "-C", root, "cat-file", "-e", ref+"^{commit}").Run() == nil
}

func mergeBaseExists(ctx context.Context, root, base, head string) bool {
	return exec.CommandContext(ctx, "git", "-C", root, "merge-base", base, head).Run() == nil
}

func gitFetch(ctx context.Context, root, remote string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root, "fetch", remote}, args...)...)
	var diagnostic strings.Builder
	cmd.Stderr = &diagnostic
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "fetch %s %s: %s", remote, strings.Join(args, " "), strings.TrimSpace(diagnostic.String()))
	}
	return nil
}
