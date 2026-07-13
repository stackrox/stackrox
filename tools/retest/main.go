package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
)

var allowedCheckFailurePrefixes = []string{
	"codecov/",
}

var retestableCheckPrefixes = []string{
	"e2e-",
}

func hasAnyPrefix(name string, prefixes []string) bool {
	return slices.ContainsFunc(prefixes, func(prefix string) bool {
		return strings.HasPrefix(name, prefix)
	})
}

// skipReason explains why a job (or, when job is empty, the PR's overall
// "/retest") was not retested. Decision functions return skipReasons instead
// of logging directly, so that they stay pure and unit-testable, and so that
// every "why won't this be retested" message is formatted consistently by a
// single call site that has the PR number in scope.
type skipReason struct {
	job     string
	message string
}

func logSkipReason(prNumber int, reason skipReason) {
	if reason.job == "" {
		log.Printf("#%d not issuing /retest: %s", prNumber, reason.message)
		return
	}
	log.Printf("#%d not issuing /test %q: %s", prNumber, reason.job, reason.message)
}

const s = "stackrox"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))
	if err := run(ctx, client); err != nil {
		log.Fatal(err.Error())
	}
}

func run(ctx context.Context, client *github.Client) error {
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("could not get current user: %w", err)
	}
	log.Printf("Logged as %s: %s", user.GetLogin(), user.GetHTMLURL())

	// TODO(janisz): handle pagination
	search, _, err := client.Search.Issues(ctx, `repo:stackrox/stackrox label:auto-retest state:open type:pr`, nil)
	if err != nil {
		return fmt.Errorf("could not find issues: %w", err)
	}
	log.Printf("Found %d PRs", search.GetTotal())

issues:
	for _, pr := range search.Issues {
		prNumber := pr.GetNumber()
		log.Printf("#%d retrieving...: %s", prNumber, pr.GetHTMLURL())
		prDetails, _, err := client.PullRequests.Get(ctx, s, s, prNumber)
		if err != nil {
			log.Printf("#%d could not get PR details: %v", prNumber, err)
			continue
		}
		userComments, allComments, err := commentsForPrByUser(ctx, client, prNumber, user.GetID())
		if err != nil {
			log.Printf("#%d could not get allComments: %v", prNumber, err)
			continue
		}
		log.Printf("#%d has %d allComments by %s and %d in total", prNumber, len(userComments), user.GetLogin(), len(allComments))
		checks, err := checksForCommit(ctx, client, prDetails.GetHead().GetSHA())
		if err != nil {
			log.Printf("#%d could not get checks: %v", prNumber, err)
			continue
		}
		log.Printf("#%d has %d completed checks", prNumber, len(checks))

		skippableCheckPrefixes := slices.Concat(allowedCheckFailurePrefixes, retestableCheckPrefixes)
		for name, state := range checks {
			if state != jobFailure || hasAnyPrefix(name, skippableCheckPrefixes) {
				continue
			}
			log.Printf("#%d has a failing check (%s), skipping", prNumber, name)
			continue issues
		}

		statuses, err := statusesForPR(ctx, client, prDetails.GetStatusesURL())
		if err != nil {
			log.Printf("#%d could not get statuses: %v", prNumber, err)
			continue
		}
		log.Printf("#%d has %d statuses", prNumber, len(statuses))
		jobsToRetest, jobSkips, err := jobsToRetestFromComments(userComments, allComments)
		if err != nil {
			log.Printf("#%d could not get jobs to retest: %v", prNumber, err)
			for _, c := range userComments {
				if c == err.Error() {
					continue issues
				}
			}
			errorComment := fmt.Sprintf(":x: There was an error with a comment. "+
				"Please edit or remove it and issue a proper command\n%s", err.Error())
			createComment(ctx, client, prNumber, errorComment)
			continue
		}
		for _, skip := range jobSkips {
			logSkipReason(prNumber, skip)
		}
		log.Printf("#%d jobs to retest: %s", prNumber, strings.Join(jobsToRetest, ", "))
		retestReason := skipRetestReason(statuses, userComments, checks)
		if retestReason != nil {
			logSkipReason(prNumber, skipReason{message: retestReason.Error()})
		}
		newComments, testSkips := commentsToCreate(statuses, jobsToRetest, retestReason == nil)
		for _, skip := range testSkips {
			logSkipReason(prNumber, skip)
		}
		createComment(ctx, client, prNumber, strings.Join(newComments, "\n"))
	}
	return nil
}

var (
	restestNTimes = regexp.MustCompile(`/retest-times\s+(\d+)\s+(.*)`)
	testJob       = regexp.MustCompile(`/test\s+(.*)`)
)

func commentsToCreate(statuses map[string]jobState, jobsToRetest []string, shouldRetest bool) (comments []string, skipped []skipReason) {
	for _, job := range jobsToRetest {
		if statuses[job] == jobPending {
			skipped = append(skipped, skipReason{job: job, message: "already pending"})
			continue
		}
		comments = append(comments, "/test "+job)
	}

	if len(jobsToRetest) != 0 || !shouldRetest {
		return comments, skipped
	}
	comments = append(comments, retestComment)
	return comments, skipped
}

func jobsToRetestFromComments(userComments, allComments []string) ([]string, []skipReason, error) {
	testedJobs := map[string]int{}
	for _, c := range userComments {
		testJobMatch := testJob.FindStringSubmatch(c)
		if len(testJobMatch) == 2 {
			job := strings.TrimSpace(testJobMatch[1])
			if _, ok := testedJobs[job]; !ok {
				testedJobs[job] = 0
			}
			testedJobs[job]++
			continue
		}
	}

	requestedRetests := map[string]int{}
	for _, c := range allComments {
		matched := restestNTimes.FindStringSubmatch(c)
		if len(matched) != 3 {
			continue
		}
		job := strings.TrimSpace(matched[2])
		t, err := strconv.Atoi(matched[1])
		if err != nil {
			return nil, nil, fmt.Errorf("got an error in a comment %q: %w", c, err)
		}
		if t < 1 || t > 100 {
			return nil, nil, fmt.Errorf("invalid retest number requested: %q", c)
		}
		if _, ok := requestedRetests[job]; !ok {
			requestedRetests[job] = 0
		}
		requestedRetests[job] += t
	}

	jobsToRetest := make([]string, 0, len(requestedRetests))
	var skipped []skipReason
	for job, requested := range requestedRetests {
		remaining := requested - testedJobs[job]
		if remaining < 1 {
			skipped = append(skipped, skipReason{
				job:     job,
				message: fmt.Sprintf("exceeded retest budget of %d, already tested %d times", requested, testedJobs[job]),
			})
			continue
		}
		jobsToRetest = append(jobsToRetest, job)
	}
	slices.Sort(jobsToRetest)
	slices.SortFunc(skipped, func(a, b skipReason) int { return strings.Compare(a.job, b.job) })

	return jobsToRetest, skipped, nil
}

const retestComment = "/retest"

// skipRetestReason reports why a "/retest" comment should not be issued.
// It returns nil when a retest is warranted, and a human-readable reason otherwise.
func skipRetestReason(statuses map[string]jobState, comments []string, checks map[string]jobState) error {
	retested := 0
	for _, c := range comments {
		if c == retestComment {
			retested++
		}
	}
	if retested > 3 {
		return fmt.Errorf("PR has already been retested %d times", retested)
	}

	for name, state := range checks {
		if state == jobFailure && hasAnyPrefix(name, retestableCheckPrefixes) {
			return nil
		}
	}

	for _, state := range statuses {
		if state == jobFailure {
			return nil
		}
	}
	return errors.New("no failing status or retestable check found")
}
