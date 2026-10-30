package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const enforceLabel = "ci-dispatcher-enforce"

type pullRequestEvent struct {
	PullRequest *struct {
		Labels []struct{ Name string }
		Base   struct{ SHA string }
		Head   struct{ SHA string }
	} `json:"pull_request"`
}

// Why the selector computes the diff itself instead of taking a file list:
//
//   - Decisions are only as trustworthy as the file list. The pull request
//     files API truncates silently above 3000 files, and a truncated list can
//     look like complete skip coverage, suppressing tests that must run. Git
//     has no partial-success mode: the diff either runs or errors here.
//   - Exactness has three load-bearing details that must stay with the diff:
//     --no-renames counts both sides of a rename, NUL separation keeps
//     newline-bearing paths whole, and the three-dot range measures from the
//     merge base. An external producer must replicate all three or skip
//     decisions silently drift.
//   - Fail-closed is local. Any error here returns to ghaDecision, which
//     turns it into run-everything. An external producer would rebuild that
//     contract at a hand-off boundary instead of reusing it.
//   - The cheap exact input is a shallow fetch of just the two event SHAs,
//     deepened only until the merge base resolves. Full history
//     (fetch-depth: 0) is exact but re-fetches the repository on every pull
//     request event; a fixed shallow depth is cheap but silently loses
//     selection for older pull requests.
//   - The binary already parses the event JSON for labels and SHAs. An
//     external producer adds a second parser, in bash, duplicated across
//     callers, and none of the diff semantics remain covered by go test.
//     Complete repositories, including every test fixture, contain the merge
//     base already and fetch nothing.
func changedFiles(ctx context.Context, root, base, head string) ([]string, error) {
	if base == "" || head == "" || strings.HasPrefix(base, "-") || strings.HasPrefix(head, "-") {
		return nil, errors.New("missing or invalid diff refs")
	}
	if err := fetchDiffHistory(ctx, root, base, head, "origin"); err != nil {
		return nil, err
	}
	// Disabling rename detection includes both paths, so moving a backend
	// fixture into docs cannot turn the removal into a docs-only change.
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

func ghaDecision(ctx context.Context, root string) (decision, bool) {
	if os.Getenv("GITHUB_EVENT_NAME") != "pull_request" {
		return fallback(errors.New("selection applies only to pull requests")), false
	}
	data, err := os.ReadFile(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil {
		return fallback(errors.Wrap(err, "read PR event")), false
	}
	var event pullRequestEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fallback(errors.Wrap(err, "decode PR event")), false
	}
	if event.PullRequest == nil {
		return fallback(errors.New("event has no pull request")), false
	}
	pr := event.PullRequest
	enforce := slices.ContainsFunc(pr.Labels, func(label struct{ Name string }) bool { return label.Name == enforceLabel })
	m, err := loadMapping(filepath.Join(root, "ci/test-selection-rules.yaml"))
	if err != nil {
		return fallback(err), enforce
	}
	files, err := changedFiles(ctx, root, pr.Base.SHA, pr.Head.SHA)
	if err != nil {
		return fallback(err), enforce
	}
	return resolve(files, m), enforce
}

func gha(ctx context.Context, output, diagnostic io.Writer) error {
	root := os.Getenv("CI_SELECT_ROOT")
	if root == "" {
		root = "."
	}
	d, enforce := ghaDecision(ctx, root)
	var details []string
	applied := make(map[string]string, len(d.actions))
	for _, target := range slices.Sorted(maps.Keys(d.actions)) {
		applied[target] = "run"
		if enforce {
			applied[target] = d.action(target)
		}
		details = append(details, fmt.Sprintf("%s: planned=%s; applied=%s\n", target, d.action(target), applied[target]))
	}
	// Section headers keep the step summary readable: the decisions list and
	// the trace are separated from each other and from the enforce flag.
	body := fmt.Sprintf("enforce=%t\n-- decisions --\n%s\n-- trace --\n%s%s",
		enforce, d.String(), d.log, strings.Join(details, ""))
	data, err := json.Marshal(applied)
	if err != nil {
		return errors.Wrap(err, "encode decisions")
	}
	if err := appendFile(os.Getenv("GITHUB_OUTPUT"), "decisions="+string(data)+"\n"); err != nil {
		return err
	}
	fmt.Fprint(output, body)
	return appendFile(os.Getenv("GITHUB_STEP_SUMMARY"), "## CI selection\n\n<pre>"+html.EscapeString(body)+"</pre>\n")
}

func appendFile(path, text string) error {
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return errors.Wrap(err, "open CI output")
	}
	_, writeErr := io.WriteString(file, text)
	closeErr := file.Close()
	if writeErr != nil {
		return errors.Wrap(writeErr, "write CI output")
	}
	return errors.Wrap(closeErr, "close CI output")
}
