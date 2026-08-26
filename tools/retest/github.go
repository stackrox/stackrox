package main

import (
	"context"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
)

func createComment(ctx context.Context, client *github.Client, prNumber int, comment string) {
	if comment == "" {
		log.Printf("#%d not commented", prNumber)
		return
	}
	log.Printf("#%d will be commented with: %s", prNumber, comment)
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
	var allComments, userComments []string
	nextPage := 0
	for {
		comments, resp, err := client.Issues.ListComments(ctx, s, s, prNumber, &github.IssueListCommentsOptions{
			Sort:        new("created"),
			Direction:   new("asc"),
			ListOptions: github.ListOptions{Page: nextPage},
		})
		if err != nil {
			return nil, nil, err
		}
		for _, comment := range comments {
			c := splitMultilineComment(comment.GetBody())
			if comment.User.GetID() == userId {
				userComments = append(userComments, c...)
			}
			allComments = append(allComments, c...)
		}
		if resp.NextPage == 0 {
			return userComments, allComments, nil
		}
		nextPage = resp.NextPage
	}
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

// jobState normalizes a job's outcome regardless of which GitHub API it
// came from: the Statuses API (Prow jobs, reported as free-form strings
// such as "success"/"failure"/"pending"/"error") and the Checks API
// (GitHub Actions, reported as a pass/fail conclusion) each have their own
// raw representation. Decision logic in main.go only ever needs to ask "is
// this job failing?" or "is this job still pending?", so both sources are
// translated into this one type at the point they're fetched, instead of
// letting two different truthiness conventions leak into the rest of the
// file. A status of "error" (the job didn't reach a verdict, e.g. due to
// infra trouble) is folded into jobFailure alongside "failure" (the job
// reached a verdict and it was negative), since both warrant a retest the
// same way. The zero value, jobOK, covers every other raw state (success,
// cancelled, or simply "no news") since none of those get special
// treatment today.
type jobState int

const (
	jobOK jobState = iota
	jobPending
	jobFailure
)

func checksForCommit(ctx context.Context, client *github.Client, lastCommit string) (map[string]jobState, error) {
	completed := "completed"
	latest := "latest"
	checks, _, err := client.Checks.ListCheckRunsForRef(ctx, s, s, lastCommit, &github.ListCheckRunsOptions{
		Status: &completed,
		Filter: &latest,
	})
	if err != nil {
		return nil, err
	}

	result := map[string]jobState{}
	for _, check := range checks.CheckRuns {
		result[check.GetName()] = checkToState(check)
	}
	return result, nil
}

func checkToState(check *github.CheckRun) jobState {
	// "timed_out" means the check never reached a verdict, same as a
	// Statuses API "error". "error" is folded into jobFailure alongside
	// "failure" so retestable checks (e.g. "e2e-...") still trigger a retry.
	//
	// "cancelled" is mapped to jobOK: because `retest_comment.yml` only
	// reruns GitHub Actions runs with status=failure, so issuing "/retest"
	// for a cancelled run wouldn't trigger anything anyway.
	switch check.GetConclusion() {
	case "failure", "timed_out":
		return jobFailure
	default:
		return jobOK
	}
}

type Status struct {
	Context   string    `json:"context"`
	State     string    `json:"state"`
	UpdatedAt time.Time `json:"updated_at"`
}

func statusesForPR(ctx context.Context, client *github.Client, url string) (map[string]jobState, error) {
	var statuses []Status
	statusRequest, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	_, err = client.Do(ctx, statusRequest, &statuses)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(statuses, func(a, b Status) int {
		return a.UpdatedAt.Compare(b.UpdatedAt)
	})

	result := map[string]jobState{}
	for _, status := range statuses {
		job := strings.TrimPrefix(status.Context, "ci/prow/")
		result[job] = parseJobState(status.State)
	}

	return result, nil
}

func parseJobState(raw string) jobState {
	switch raw {
	case "failure", "error":
		return jobFailure
	case "pending":
		return jobPending
	default:
		return jobOK
	}
}
