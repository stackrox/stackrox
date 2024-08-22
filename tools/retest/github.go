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
			Sort:        github.String("created"),
			Direction:   github.String("asc"),
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
	Context   string    `json:"context"`
	State     string    `json:"state"`
	UpdatedAt time.Time `json:"updated_at"`
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

	slices.SortFunc(statuses, func(a, b Status) int {
		return a.UpdatedAt.Compare(b.UpdatedAt)
	})

	result := map[string]string{}
	for _, status := range statuses {
		job := strings.TrimPrefix(status.Context, "ci/prow/")
		result[job] = status.State
	}

	return result, nil
}
