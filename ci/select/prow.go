package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var testLine = regexp.MustCompile(`^/test\b(.*)$`)

type comment struct {
	CreatedAt string `json:"created_at"`
	Body      string `json:"body"`
}

func commentRequests(job, body string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		match := testLine.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}
		names := strings.Fields(match[1])
		if slices.Contains(names, job) || slices.Contains(names, "all") {
			return true
		}
	}
	return false
}

func requestsSince(comments []comment, job, commitTime string) bool {
	commit, err := time.Parse(time.RFC3339Nano, commitTime)
	if err != nil {
		return false
	}
	for _, comment := range comments {
		if !commentRequests(job, comment.Body) {
			continue
		}
		created, err := time.Parse(time.RFC3339Nano, comment.CreatedAt)
		if err == nil && !created.Before(commit) {
			return true
		}
	}
	return false
}

func readComments(path string) ([]comment, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read comments file")
	}
	comments := []comment{}
	if err := json.Unmarshal(text, &comments); err != nil {
		return nil, errors.Wrap(err, "decode comments file")
	}
	if comments == nil {
		return nil, fmt.Errorf("comments file must be a list")
	}
	return comments, nil
}

type pullRequest struct {
	Number int    `json:"number"`
	Body   string `json:"body"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Base struct {
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
}

func prowCI(ctx context.Context, job string, input io.Reader, diagnostic io.Writer) (string, error) {
	var pr pullRequest
	if err := json.NewDecoder(input).Decode(&pr); err != nil {
		return "run", fmt.Errorf("read PR details: %w", err)
	}
	meta := metadata{repo: pr.Base.Repo.FullName, pr: strconv.Itoa(pr.Number)}
	if _, err := repositoryPath(meta.repo, meta.pr); err != nil {
		return "run", err
	}
	labels := make([]string, 0, len(pr.Labels))
	for _, label := range pr.Labels {
		labels = append(labels, label.Name)
	}
	enforced := slices.Contains(labels, enforceLabel)
	if meta.repo == "openshift/release" {
		// Rehearsal PRs carry the requested label in their description.
		enforced = regexp.MustCompile(`/label:\s*ci-dispatcher-enforce`).MatchString(pr.Body)
	}
	files, unknown := changedFiles(ctx, os.Getenv("PULL_BASE_SHA"), "HEAD")
	head := envOr("PULL_PULL_SHA", "HEAD")
	commit, err := gitOutput(ctx, "rev-parse", "--short=7", head)
	if err != nil {
		commit = "unknown"
	}
	meta.commit = commit
	// Prow's synthetic merge can be newer than a /test comment on the PR head.
	commitTime, _ := gitOutput(ctx, "show", "-s", "--format=%cI", head)
	comments, err := newGitHubClient().comments(ctx, meta.repo, meta.pr)
	if err != nil {
		return "run", fmt.Errorf("read /test comments; job will run: %w", err)
	}
	rules, defaultsPath := rulePaths()
	resolved, d, err := openDecision(rules, defaultsPath, files, labels, unknown, meta)
	if err != nil {
		return "run", err
	}
	fmt.Fprint(diagnostic, resolved.log)
	forced := requestsSince(comments, job, commitTime)
	if forced {
		fmt.Fprintf(diagnostic, "a /test comment runs %s\n", job)
	}
	action := gateAction(job, resolved.jobs, d, enforced, forced)
	fmt.Fprintf(diagnostic, "%s: %s (enforce=%t)\n", job, action, enforced)
	return action, nil
}
