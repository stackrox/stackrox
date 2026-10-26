// Command timing-report measures observed PR workflow jobs and maintains one PR comment.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"time"
)

const commentMarker = "<!-- stackrox-native-ci-timing:v1 -->"

var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type (
	// requestFunc isolates authenticated API calls from timing and comment logic.
	requestFunc func(context.Context, string, string, []byte) ([]byte, error)

	pullRequest struct {
		Head struct {
			SHA  string `json:"sha"`
			Ref  string `json:"ref"`
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"head"`
	}

	workflowRun struct {
		Attempt    int       `json:"run_attempt"`
		Event      string    `json:"event"`
		HeadSHA    string    `json:"head_sha"`
		HeadBranch string    `json:"head_branch"`
		Conclusion string    `json:"conclusion"`
		CreatedAt  time.Time `json:"created_at"`
		HeadRepo   struct {
			FullName string `json:"full_name"`
		} `json:"head_repository"`
		PullRequests []struct {
			Number int `json:"number"`
		} `json:"pull_requests"`
	}

	workflowJob struct {
		ID          int64          `json:"id"`
		Name        string         `json:"name"`
		Status      string         `json:"status"`
		Conclusion  string         `json:"conclusion"`
		CompletedAt time.Time      `json:"completed_at"`
		Steps       []workflowStep `json:"steps"`
	}
	workflowStep struct {
		Name       string `json:"name"`
		Conclusion string `json:"conclusion"`
	}

	issueComment struct {
		ID      int64  `json:"id"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		User    struct {
			Login string `json:"login"`
		} `json:"user"`
	}
)

func main() {
	output, diagnostic := os.Stdout, os.Stderr //nolint:forbidigo // This command prints Markdown and API diagnostics to separate streams.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args[1:], ghRequest, output, diagnostic); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(diagnostic, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, request requestFunc, output, diagnostic io.Writer) error {
	flags := flag.NewFlagSet("timing-report", flag.ContinueOnError)
	flags.SetOutput(diagnostic)
	repo := flags.String("repo", "stackrox/stackrox", "GitHub owner/repository")
	pr := flags.Int("pr", 0, "pull request number (required)")
	runID := flags.Int64("run", 0, "initial workflow run ID (required)")
	post := flags.Bool("post", false, "publish using the GitHub Actions bot token (Actions only)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parse arguments: %w", err)
	}
	if flags.NArg() != 0 || *pr <= 0 || *runID <= 0 || !repositoryPattern.MatchString(*repo) {
		return errors.New("supply positive --pr and --run numbers and --repo as owner/repository; positional arguments are not supported")
	}
	if *post && (os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("GITHUB_TOKEN") == "") {
		return errors.New("publishing is only supported in GitHub Actions with its GITHUB_TOKEN; local execution is preview-only")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("stop timing report: %w", err)
	}
	return refresh(ctx, request, *repo, *pr, *runID, *post, output, diagnostic)
}

// ghRequest uses the workflow token in Actions and gh authentication for local previews.
func ghRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	args := []string{"api", "--hostname", "github.com", "--method", method, path}
	if method == "GET" && strings.HasSuffix(path, "/logs") {
		args = append(args, "--allow-escape-sequences", "--header", "Range: bytes=0-1048575")
	} else if method == "GET" {
		args = append(args, "--paginate", "--slurp")
	} else {
		args = append(args, "--input", "-")
	}
	cmd := exec.CommandContext(callCtx, "gh", args...)
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		cmd.Env = append(os.Environ(), "GH_TOKEN="+os.Getenv("GITHUB_TOKEN"))
	}
	cmd.Stdin = bytes.NewReader(body)
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	data, err := cmd.Output()
	if err != nil {
		if callCtx.Err() != nil {
			return nil, fmt.Errorf("GitHub request %s: %w", path, callCtx.Err())
		}
		return nil, fmt.Errorf("gh api %s: %w: %s", path, err, strings.TrimSpace(diagnostic.String()))
	}
	return data, nil
}

// getPages decodes gh --paginate --slurp, including endpoints that return only one page.
func getPages[T any](ctx context.Context, request requestFunc, path string) ([]T, error) {
	data, err := request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var pages []T
	if err := json.Unmarshal(data, &pages); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return pages, nil
}

func refresh(ctx context.Context, request requestFunc, repo string, pr int, runID int64, post bool, output, diagnostic io.Writer) error {
	prefix := "repos/" + repo
	prPath := fmt.Sprintf("%s/pulls/%d", prefix, pr)
	prs, err := getPages[pullRequest](ctx, request, prPath)
	if err != nil {
		return err
	}
	if len(prs) != 1 || prs[0].Head.SHA == "" {
		return errors.New("expected one pull request with a head commit")
	}
	current, err := collect(ctx, request, prefix, pr, prs[0], runID)
	if err != nil {
		return err
	}
	if current.SHA == "" {
		fmt.Fprintln(diagnostic, "Cancelled workflow: timing contribution excluded")
		return nil
	}
	commentsPath := fmt.Sprintf("%s/issues/%d/comments", prefix, pr)
	pages, err := getPages[[]issueComment](ctx, request, commentsPath+"?per_page=100")
	if err != nil {
		return err
	}
	var existing issueComment
	for _, page := range pages {
		for _, comment := range page {
			if comment.User.Login == "github-actions[bot]" && strings.HasPrefix(comment.Body, commentMarker+"\n") {
				if existing.ID != 0 {
					return errors.New("multiple timing comments by github-actions[bot]; refusing to choose one to overwrite")
				}
				existing = comment
			}
		}
	}
	body, err := render(repo, existing.Body, current)
	if err != nil {
		return err
	}
	if len(body) > 60000 {
		return errors.New("timing comment exceeds 60,000 bytes; refusing to truncate history")
	}
	if !post {
		if _, err := fmt.Fprintln(output, body); err != nil {
			return fmt.Errorf("write timing preview: %w", err)
		}
		return nil
	}
	if body == existing.Body {
		fmt.Fprintln(diagnostic, "Timing comment unchanged: "+existing.HTMLURL)
		return nil
	}
	method, path := "POST", commentsPath
	if existing.ID != 0 {
		method, path = "PATCH", fmt.Sprintf("%s/issues/comments/%d", prefix, existing.ID)
	}
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return fmt.Errorf("encode timing comment: %w", err)
	}
	data, err := request(ctx, method, path, payload)
	if err != nil {
		return err
	}
	var comment issueComment
	if err := json.Unmarshal(data, &comment); err != nil {
		return fmt.Errorf("decode published comment: %w", err)
	}
	fmt.Fprintln(diagnostic, "Timing comment updated: "+comment.HTMLURL)
	return nil
}

// belongsToPR handles omitted PR associations only when both source repository and branch match.
func belongsToPR(run workflowRun, pr int, head pullRequest) bool {
	if len(run.PullRequests) == 0 {
		return run.HeadBranch == head.Head.Ref && run.HeadRepo.FullName == head.Head.Repo.FullName
	}
	return slices.ContainsFunc(run.PullRequests, func(ref struct {
		Number int `json:"number"`
	}) bool {
		return ref.Number == pr
	})
}

// collect reads only the caller's first attempt; native workflow conditions exclude label events and reruns.
func collect(ctx context.Context, request requestFunc, prefix string, pr int, head pullRequest, runID int64) (measurement, error) {
	path := fmt.Sprintf("%s/actions/runs/%d/attempts/1", prefix, runID)
	runs, err := getPages[workflowRun](ctx, request, path)
	if err != nil {
		return measurement{}, err
	}
	if len(runs) != 1 || runs[0].Event != "pull_request" || runs[0].Attempt != 1 || !belongsToPR(runs[0], pr, head) {
		return measurement{}, errors.New("expected the PR's initial workflow attempt")
	}
	if runs[0].Conclusion == "cancelled" {
		return measurement{}, nil
	}
	pages, err := getPages[struct {
		Jobs []workflowJob `json:"jobs"`
	}](ctx, request, path+"/jobs?per_page=100")
	if err != nil {
		return measurement{}, err
	}
	var jobs []workflowJob
	for _, page := range pages {
		jobs = append(jobs, page.Jobs...)
	}
	current, err := measure(runs[0], jobs)
	if err != nil {
		return measurement{}, err
	}
	current.Reports = []selectionReport{collectDecisions(ctx, request, prefix, runID, jobs)}
	return current, nil
}
