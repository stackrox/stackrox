package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"

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

loop:
	for _, pr := range search.Issues {
		prNumber := pr.GetNumber()
		log.Printf("#%d processing...", prNumber)
		prDetails, _, err := client.PullRequests.Get(ctx, S, S, prNumber)
		handleError(err)
		commentsBodies := commentsForPR(ctx, client, prNumber)
		checks := checksForCommit(ctx, client, prDetails.GetHead().GetSHA())

		for name, status := range checks {
			if !status {
				log.Printf("#%d has a failing check (%s) skipping", prNumber, name)
				continue loop
			}
		}

		statuses := statusForPR(ctx, client, prDetails.GetStatusesURL())
		jobsToRetest := jobsToRetestFromComments(commentsBodies)

		for _, job := range jobsToRetest {
			status := statuses[job]
			if status.State != "pending" {
				continue
			}
			log.Printf("#%d retesting: %s %s", prNumber, job, status.State)
			comment := github.IssueComment{
				Body: ptr("/test " + job),
			}
			testComment, _, err := client.Issues.CreateComment(ctx, S, S, prNumber, &comment)
			handleError(err)
			log.Printf("#%d commented: %s", prNumber, testComment.GetURL())
		}

		if len(jobsToRetest) != 0 {
			continue
		}

		retestComment := github.IssueComment{
			Body: ptr(retestComment),
		}
		if !shouldRetest(statuses, commentsBodies) {
			continue
		}
		comment, _, err := client.Issues.CreateComment(ctx, S, S, prNumber, &retestComment)
		handleError(err)
		log.Printf("#%d commented: %s", prNumber, comment.GetURL())
	}
}

var (
	restestNTimes = regexp.MustCompile("Retest (.*) (\\d+) times")
	testJob       = regexp.MustCompile("/test (.*)")
)

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
		job := matched[1]
		t, err := strconv.Atoi(matched[2])
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

func shouldRetest(statuses map[string]Status, comments []string) bool {

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
		if status.State == "failure" {
			return true
		}
	}
	return false
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

func statusForPR(ctx context.Context, client *github.Client, url string) map[string]Status {
	var statuses []Status
	statusRequest, err := http.NewRequest("GET", url, nil)
	handleError(err)
	_, err = client.Do(ctx, statusRequest, &statuses)
	handleError(err)

	result := map[string]Status{}
	for _, status := range statuses {
		result[status.Context] = status
	}

	return result
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func ptr[T any](in T) *T {
	return &in
}
