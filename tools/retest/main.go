package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
)

const (
	s              = "stackrox"
	githubTokenEnv = "GITHUB_TOKEN"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	// Use installation transport with client.
	client := github.NewClient(nil).WithAuthToken(os.Getenv(githubTokenEnv))
	if err := run(ctx, client); err != nil {
		log.Fatalf(err.Error())
	}
}

func run(ctx context.Context, client *github.Client) error {
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("could not get user: %w", err)
	}
	log.Printf("Logged as %s: %s", user.GetLogin(), user.GetHTMLURL())

	// TODO(janisz): handle pagination
	search, _, err := client.Search.Issues(ctx, `repo:stackrox/stackrox label:auto-retest state:open type:pr status:failure`, nil)
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

		for name, status := range checks {
			if !status {
				log.Printf("#%d has a failing check (%s), skipping", prNumber, name)
				continue issues
			}
		}

		statuses, err := statusesForPR(ctx, client, prDetails.GetStatusesURL())
		if err != nil {
			log.Printf("#%d could not get statuses: %v", prNumber, err)
			continue
		}
		log.Printf("#%d has %d statuses", prNumber, len(statuses))
		jobsToRetest, err := jobsToRetestFromComments(userComments, allComments)
		if err != nil {
			log.Printf("#%d could not get jobs to retest: %v", prNumber, err)
			for _, c := range userComments {
				if c == err.Error() {
					continue issues
				}
			}
			errorComment := fmt.Sprintf(":x: There was an error with a comment. "+
				"Please eddit or remove it and issue a proper command\n%s", err.Error())
			createComment(ctx, client, prNumber, errorComment)
			continue
		}
		log.Printf("#%d jobs to retest: %s", prNumber, strings.Join(jobsToRetest, ", "))
		newComments := commentsToCreate(statuses, jobsToRetest, shouldRetestFailedStatuses(statuses, userComments))
		log.Printf("#%d will be commented with: %s", prNumber, strings.Join(newComments, ", "))
		createComment(ctx, client, prNumber, strings.Join(newComments, "\n"))
	}
	return nil
}

var (
	restestNTimes = regexp.MustCompile(`/retest-times (\d+) (.*)`)
	testJob       = regexp.MustCompile(`/test (.*)`)
)

func commentsToCreate(statuses map[string]string, jobsToRetest []string, shouldRetest bool) []string {
	var comments []string
	for _, job := range jobsToRetest {
		state := statuses[job]
		if state == "pending" {
			continue
		}
		comments = append(comments, "/test "+job)
	}

	if len(jobsToRetest) != 0 {
		return comments
	}

	if !shouldRetest {
		return comments
	}
	comments = append(comments, retestComment)
	return comments
}

func jobsToRetestFromComments(userComments, allComments []string) ([]string, error) {
	testedJobs := map[string]int{}
	for _, c := range userComments {
		testJobMatch := testJob.FindStringSubmatch(c)
		if len(testJobMatch) == 2 {
			job := testJobMatch[1]
			if _, ok := testedJobs[job]; !ok {
				testedJobs[job] = 0
			}
			testedJobs[job]++
			continue
		}
	}

	jobsToRetest := map[string]int{}
	for _, c := range allComments {
		matched := restestNTimes.FindStringSubmatch(c)
		if len(matched) != 3 {
			continue
		}
		job := matched[2]
		t, err := strconv.Atoi(matched[1])
		if err != nil {
			return nil, fmt.Errorf("got an error in a comment %q: %w", c, err)
		}
		if t < 1 || t > 100 {
			return nil, fmt.Errorf("invalid retest number requested: %q", c)
		}
		if _, ok := jobsToRetest[job]; !ok {
			jobsToRetest[job] = 0
		}
		jobsToRetest[job] += t
	}

	missingTests := make([]string, 0, len(jobsToRetest))
	for job, times := range jobsToRetest {
		toTest := times - testedJobs[job]
		if toTest < 1 {
			continue
		}
		missingTests = append(missingTests, job)
	}
	slices.Sort(missingTests)

	return missingTests, nil
}

const retestComment = "/retest"

func shouldRetestFailedStatuses(statuses map[string]string, comments []string) bool {
	retested := 0
	for _, c := range comments {
		if c == retestComment {
			retested++
		}
	}
	if retested > 3 {
		return false
	}

	for _, status := range statuses {
		if status == "failure" {
			return true
		}
	}
	return false
}

//region Github client helper

func createComment(ctx context.Context, client *github.Client, prNumber int, comment string) {
	issueComment := &github.IssueComment{
		Body: &comment,
	}
	c, _, err := client.Issues.CreateComment(ctx, s, s, prNumber, issueComment)
	if err != nil {
		log.Printf("#%d could not create a comment: %v", prNumber, err)
		return
	}
	log.Printf("#%d commented: %s", prNumber, c.GetHTMLURL())
}

func commentsForPrByUser(ctx context.Context, client *github.Client, prNumber int, userId int64) ([]string, []string, error) {
	comments, _, err := client.Issues.ListComments(ctx, s, s, prNumber, nil)
	if err != nil {
		return nil, nil, err
	}
	userComments := make([]string, 0, len(comments))
	allComments := make([]string, 0, len(comments))
	for _, comment := range comments {
		c := splitMultilineComment(comment.GetBody())
		if comment.User.GetID() == userId {
			userComments = append(userComments, c...)
		}
		allComments = append(allComments, c...)
	}
	return userComments, allComments, nil
}

func splitMultilineComment(comment string) []string {
	split := strings.Split(comment, "\n")
	result := make([]string, 0, len(split))
	for _, c := range split {
		trimmed := strings.TrimSpace(c)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func checksForCommit(ctx context.Context, client *github.Client, lastCommit string) (map[string]bool, error) {
	completed := "completed"
	latest := "latest"
	checks, _, err := client.Checks.ListCheckRunsForRef(ctx, s, s, lastCommit, &github.ListCheckRunsOptions{
		Status: &completed,
		Filter: &latest,
	})
	if err != nil {
		return nil, err
	}

	result := map[string]bool{}
	for _, check := range checks.CheckRuns {
		result[check.GetName()] = check.GetConclusion() != "failure"
	}
	return result, nil
}

type Status struct {
	Context string `json:"context"`
	State   string `json:"state"`
}

func statusesForPR(ctx context.Context, client *github.Client, url string) (map[string]string, error) {
	var statuses []Status
	statusRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	_, err = client.Do(ctx, statusRequest, &statuses)
	if err != nil {
		return nil, err
	}

	result := map[string]string{}
	for _, status := range statuses {
		job := strings.TrimPrefix(status.Context, "ci/prow/")
		result[job] = status.State
	}

	return result, nil
}

// endregion
