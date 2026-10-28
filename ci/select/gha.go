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
	"slices"
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

func changedFiles(ctx context.Context, root, base, head string) ([]string, error) {
	if base == "" || head == "" || strings.HasPrefix(base, "-") || strings.HasPrefix(head, "-") {
		return nil, errors.New("missing or invalid diff refs")
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
	m, err := loadMapping(filepath.Join(root, "ci/test-domains.yaml"))
	if err != nil {
		return fallback(err), enforce
	}
	files, err := changedFiles(ctx, root, pr.Base.SHA, pr.Head.SHA)
	if err != nil {
		return fallback(err), enforce
	}
	return resolve(files, m), enforce
}

func gha(ctx context.Context, job string, output, diagnostic io.Writer) error {
	root := os.Getenv("CI_SELECT_ROOT")
	if root == "" {
		root = "."
	}
	d, enforce := ghaDecision(ctx, root)
	body := fmt.Sprintf("enforce=%t\n%s\n%s", enforce, d.String(), d.log)
	if job != "" {
		action := "run"
		if enforce {
			action = d.action(job)
		}
		body += fmt.Sprintf("%s: planned=%s; applied=%s\n", job, d.action(job), action)
		if err := appendFile(os.Getenv("GITHUB_OUTPUT"), "action="+action+"\n"); err != nil {
			return err
		}
		fmt.Fprintln(output, "action="+action)
		fmt.Fprint(diagnostic, body)
	} else {
		applied := make(map[string]string, len(d.actions))
		for _, target := range slices.Sorted(maps.Keys(d.actions)) {
			applied[target] = "run"
			if enforce {
				applied[target] = d.action(target)
			}
			body += fmt.Sprintf("%s: planned=%s; applied=%s\n", target, d.action(target), applied[target])
		}
		data, err := json.Marshal(applied)
		if err != nil {
			return errors.Wrap(err, "encode decisions")
		}
		if err := appendFile(os.Getenv("GITHUB_OUTPUT"), "decisions="+string(data)+"\n"); err != nil {
			return err
		}
		fmt.Fprint(output, body)
	}
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
