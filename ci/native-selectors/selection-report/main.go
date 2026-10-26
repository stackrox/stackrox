// Command selection-report observes native CI decisions and maintains one bot comment.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const commentMarker = "<!-- stackrox-native-ci-timing:v1 -->"

var (
	repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	shaPattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	errNewHead  = errors.New("PR head changed; stopping this observer")
)

type (
	response struct {
		status int
		etag   string
		body   []byte
	}
	requestFunc func(context.Context, string, string, []byte, string) (response, error)
	api         struct {
		request requestFunc
		cache   map[string]response
	}
	options struct {
		repo, sha string
		pr        int
		run, job  int64
		watch     time.Duration
		interval  time.Duration
		post      bool
	}
	clock struct {
		now   func() time.Time
		sleep func(context.Context, time.Duration) error
	}
	comment struct {
		ID   int64                  `json:"id"`
		Body string                 `json:"body"`
		User struct{ Login string } `json:"user"`
	}
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args[1:], ghRequest, os.Stdout, os.Stderr); err != nil && !errors.Is(err, context.Canceled) { //nolint:forbidigo // CLI output and diagnostics use separate streams.
		fmt.Fprintln(os.Stderr, err) //nolint:forbidigo // Surface reporting failures in the job log.
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, request requestFunc, output, diagnostic io.Writer) error {
	var opt options
	flags := flag.NewFlagSet("selection-report", flag.ContinueOnError)
	flags.SetOutput(diagnostic)
	flags.StringVar(&opt.repo, "repo", "stackrox/stackrox", "GitHub owner/repository")
	flags.IntVar(&opt.pr, "pr", 0, "PR number")
	flags.StringVar(&opt.sha, "sha", "", "expected PR head commit")
	flags.Int64Var(&opt.run, "run", 0, "observer run ID, for a log link")
	flags.Int64Var(&opt.job, "job", 0, "observer job ID, for a log link")
	flags.DurationVar(&opt.watch, "watch", 0, "bounded observation window; zero previews once")
	flags.DurationVar(&opt.interval, "interval", 5*time.Second, "pause between observations")
	flags.BoolVar(&opt.post, "post", false, "update the bot comment in GitHub Actions only")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() != 0 || !repoPattern.MatchString(opt.repo) || opt.pr <= 0 || !shaPattern.MatchString(opt.sha) || opt.watch < 0 || opt.watch > 10*time.Minute || opt.interval < time.Second || opt.run < 0 || opt.job < 0 {
		return errors.New("supply --repo owner/repo, positive --pr, a full --sha, and a window between zero and 10m with interval >=1s")
	}
	if opt.post && (os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("GITHUB_TOKEN") == "" || opt.repo != "stackrox/stackrox" || opt.pr != 23602) {
		return errors.New("publication is restricted to the Actions bot on stackrox/stackrox#23602; local commands preview only")
	}
	return observe(ctx, &api{request: request, cache: map[string]response{}}, opt, clock{now: time.Now, sleep: pause}, output, diagnostic)
}

func pause(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("wait for next observation: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

// ghRequest uses native Actions credentials without exposing a personal token to workflows.
func ghRequest(ctx context.Context, method, path string, body []byte, etag string) (response, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	args := []string{"api", "--hostname", "github.com", "--include", "--method", method, path}
	if etag != "" {
		args = append(args, "--header", "If-None-Match: "+etag)
	}
	if method != "GET" {
		args = append(args, "--input", "-")
	}
	cmd := exec.CommandContext(ctx, "gh", args...)
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		cmd.Env = append(os.Environ(), "GH_TOKEN="+os.Getenv("GITHUB_TOKEN"))
	}
	cmd.Stdin = bytes.NewReader(body)
	var diagnostic bytes.Buffer
	cmd.Stderr = &diagnostic
	data, callErr := cmd.Output()
	if err := ctx.Err(); err != nil {
		return response{}, fmt.Errorf("GitHub request stopped: %w", err)
	}
	reader := bufio.NewReader(bytes.NewReader(data))
	statusLine, err := reader.ReadString('\n')
	fields := strings.Fields(statusLine)
	if err != nil || len(fields) < 2 {
		return response{}, fmt.Errorf("gh api %s: %w: %s", path, errors.Join(callErr, errors.New("missing HTTP status")), diagnostic.String())
	}
	status, err := strconv.Atoi(fields[1])
	if err != nil {
		return response{}, fmt.Errorf("read HTTP status: %w", err)
	}
	result := response{status: status}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return response{}, fmt.Errorf("read response headers: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
		key, value, _ := strings.Cut(line, ":")
		if strings.EqualFold(key, "etag") {
			result.etag = strings.TrimSpace(value)
		}
	}
	if status == 304 {
		return result, nil
	}
	if callErr != nil || status < 200 || status >= 300 {
		return response{}, fmt.Errorf("gh api %s returned HTTP %d: %s", path, status, diagnostic.String())
	}
	result.body, err = io.ReadAll(reader)
	if err != nil {
		return response{}, fmt.Errorf("read GitHub response: %w", err)
	}
	return result, nil
}

// get reuses unchanged JSON responses; authenticated conditional GETs avoid primary rate-limit charges.
func (a *api) get(ctx context.Context, path string, target any) error {
	saved := a.cache[path]
	result, err := a.request(ctx, "GET", path, nil, saved.etag)
	if err != nil {
		return err
	}
	if result.status == 304 {
		result = saved
	} else {
		a.cache[path] = result
	}
	if err := json.Unmarshal(result.body, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func observe(ctx context.Context, a *api, opt options, c clock, output, diagnostic io.Writer) error {
	deadline := c.now().Add(opt.watch)
	var saved comment
	var lastBody string
	logged := map[int64]string{}
	var rows []record
	for {
		active := opt.watch > 0 && c.now().Before(deadline)
		var err error
		if active || lastBody == "" {
			callCtx := ctx
			cancel := func() {}
			if active {
				callCtx, cancel = context.WithTimeout(ctx, deadline.Sub(c.now()))
			}
			var updated []record
			updated, err = collect(callCtx, a, opt)
			cancel()
			if err == nil {
				rows = updated
			} else if errors.Is(err, context.DeadlineExceeded) && !c.now().Before(deadline) {
				err = nil
			}
		}
		if errors.Is(err, errNewHead) {
			fmt.Fprintln(diagnostic, err)
			return nil
		}
		if err != nil {
			return err
		}
		for _, row := range rows {
			line := fmt.Sprintf("%s | %s | %s | candidate=%s eligible=%s enforced=%s | source=%s | job=%d", row.workflow, row.name, row.kind, row.candidate, row.eligible, row.enforced, row.source, row.id)
			if logged[row.id] != line {
				fmt.Fprintln(diagnostic, line)
				logged[row.id] = line
			}
		}
		active = opt.watch > 0 && c.now().Before(deadline)
		body := render(opt, rows, active, deadline)
		if opt.post && body != lastBody {
			if err := publish(ctx, a, opt, &saved, body); err != nil {
				return err
			}
		} else if !opt.post && (!active || opt.watch == 0) {
			fmt.Fprintln(output, body)
		}
		lastBody = body
		if !active {
			return nil
		}
		if err := c.sleep(ctx, min(opt.interval, deadline.Sub(c.now()))); err != nil {
			return err
		}
	}
}

func publish(ctx context.Context, a *api, opt options, saved *comment, body string) error {
	prefix := "repos/" + opt.repo + "/issues/"
	if saved.ID == 0 {
		for page := 1; ; page++ {
			var comments []comment
			if err := a.get(ctx, fmt.Sprintf("%s%d/comments?per_page=100&page=%d", prefix, opt.pr, page), &comments); err != nil {
				return err
			}
			for _, item := range comments {
				if item.User.Login == "github-actions[bot]" && strings.HasPrefix(item.Body, commentMarker+"\n") {
					if saved.ID != 0 {
						return errors.New("multiple marked bot comments; refusing to choose one")
					}
					*saved = item
				}
			}
			if len(comments) < 100 {
				break
			}
		}
	}
	if saved.Body == body {
		return nil
	}
	method, path := "POST", prefix+strconv.Itoa(opt.pr)+"/comments"
	if saved.ID != 0 {
		method, path = "PATCH", prefix+"comments/"+strconv.FormatInt(saved.ID, 10)
	}
	data, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return fmt.Errorf("encode comment: %w", err)
	}
	result, err := a.request(ctx, method, path, data, "")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(result.body, saved); err != nil {
		return fmt.Errorf("decode updated comment: %w", err)
	}
	saved.Body = body
	return nil
}

func runPath(repo, sha string, page int) string {
	return fmt.Sprintf("repos/%s/actions/runs?head_sha=%s&event=pull_request&per_page=100&page=%d", repo, url.QueryEscape(sha), page)
}
