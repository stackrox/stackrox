package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-github/v60/github"
)

const S = "stackrox"

func main() {
	ctx := context.Background()

	// Use installation transport with client.
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))

	//TODO(janisz): handle pagination
	search, _, err := client.Search.Issues(ctx, `repo:stackrox/stackrox label:auto-retest state:open type:pr status:failure`, nil)
	handleError(err)
	log.Printf("Found %d PRs", search.GetTotal())

issues:
	for _, pr := range search.Issues {
		prNumber := pr.GetNumber()
		log.Printf("#%d retrieving...", prNumber)
		prDetails, _, err := client.PullRequests.Get(ctx, S, S, prNumber)
		handleError(err)
		commentsBodies := commentsForPR(ctx, client, prNumber)
		log.Printf("#%d has %d comments", prNumber, len(commentsBodies))
		checks := checksForCommit(ctx, client, prDetails.GetHead().GetSHA())
		log.Printf("#%d has %d completed checks", prNumber, len(checks))

		for name, status := range checks {
			if !status {
				log.Printf("#%d has a failing check (%s), skipping", prNumber, name)
				continue issues
			}
		}

		statuses := statusesForPR(ctx, client, prDetails.GetStatusesURL())
		log.Printf("#%d has %d statuses", prNumber, len(statuses))
		jobsToRetest := jobsToRetestFromComments(commentsBodies)
		log.Printf("#%d jobs to retest: %s", prNumber, strings.Join(jobsToRetest, ", "))
		newComments := commentsToCreate(statuses, jobsToRetest, shouldRetest(statuses, commentsBodies))
		log.Printf("#%d will be commented with: %s", prNumber, strings.Join(newComments, ", "))
		for _, newComment := range newComments {
			createComment(ctx, client, prNumber, newComment)
		}
	}
}

var (
	restestNTimes = regexp.MustCompile("/retest-times (\\d+) (.*)")
	testJob       = regexp.MustCompile("/test (.*)")
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

func jobsToRetestFromComments(comments []string) []string {
	testedJobs := map[string]int{}
	jobsToRetest := map[string]int{}

	for _, c := range comments {
		testJobMatch := testJob.FindStringSubmatch(c)
		if len(testJobMatch) == 2 {
			job := testJobMatch[1]
			if _, ok := testedJobs[job]; !ok {
				testedJobs[job] = 0
			}
			testedJobs[job]++
			continue
		}

		matched := restestNTimes.FindStringSubmatch(c)
		if len(matched) != 3 {
			continue
		}
		job := matched[2]
		t, err := strconv.Atoi(matched[1])
		if err != nil {
			log.Printf("got an error in a comment %s: %s", c, err)
			continue
		}
		if t < 1 || t > 100 {
			log.Printf("invalid retest number requested: %s", c)
			continue
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

	return missingTests
}

const retestComment = "/retest"

func shouldRetest(statuses map[string]string, comments []string) bool {
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
	c, _, err := client.Issues.CreateComment(ctx, S, S, prNumber, issueComment)
	handleError(err)
	log.Printf("#%d commented: %s", prNumber, c.GetHTMLURL())
}

func commentsForPR(ctx context.Context, client *github.Client, prNumber int) []string {
	comments, _, err := client.Issues.ListComments(ctx, S, S, prNumber, nil)
	handleError(err)
	commentsBodies := make([]string, 0, len(comments))
	for _, comment := range comments {
		commentsBodies = append(commentsBodies, *comment.Body)
	}
	return commentsBodies
}

func checksForCommit(ctx context.Context, client *github.Client, lastCommit string) map[string]bool {
	checks, _, err := client.Checks.ListCheckRunsForRef(ctx, S, S, lastCommit, &github.ListCheckRunsOptions{
		Status: ptr("completed"),
		Filter: ptr("latest"),
	})
	handleError(err)

	result := map[string]bool{}
	for _, check := range checks.CheckRuns {
		result[check.GetName()] = check.GetConclusion() != "failure"
	}
	return result
}

type Status struct {
	Context string `json:"context"`
	State   string `json:"state"`
}

func statusesForPR(ctx context.Context, client *github.Client, url string) map[string]string {
	var statuses []Status
	statusRequest, err := http.NewRequest("GET", url, nil)
	handleError(err)
	_, err = client.Do(ctx, statusRequest, &statuses)
	handleError(err)

	result := map[string]string{}
	for _, status := range statuses {
		job := strings.TrimPrefix(status.Context, "ci/prow/")
		result[job] = status.State
	}

	return result
}

// endregion

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func ptr[T any](in T) *T {
	return &in
}
