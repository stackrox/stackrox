package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/v60/github"
)

func main() {
	ctx := context.Background()

	// Use installation transport with client.
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))

	//TODO(janisz): handle pagination
	search, _, err := client.Search.Issues(ctx, `repo:stackrox/stackrox label:auto-merge state:open type:pr status:failure`, nil)
	handleError(err)
	log.Printf("Found %d PRs", search.GetTotal())

	for _, pr := range search.Issues {
		prNumber := pr.GetNumber()
		log.Printf("Processing #%d", prNumber)
		comments, _, err := client.Issues.ListComments(ctx, "stackrox", "stackrox", prNumber, nil)
		handleError(err)

		retested := 0
		for _, c := range comments {
			if c.GetBody() == "/retest" {
				retested++
			}
		}
		log.Printf("#%d was retested %d times", prNumber, retested)
		if retested > 10 {
			continue
		}

		get, _, err := client.PullRequests.Get(ctx, "stackrox", "stackrox", prNumber)
		handleError(err)

		var statuses []Status
		request, err := http.NewRequest("GET", get.GetStatusesURL(), nil)
		handleError(err)
		_, err = client.Do(ctx, request, &statuses)
		handleError(err)

		retest := "/retest"
		retestComment := github.IssueComment{
			Body: &retest,
		}

		for _, status := range statuses {
			log.Printf("#%d %-40s\t%10s", prNumber, status.Context, status.State)
			if status.State == "failure" {
				comment, _, err := client.Issues.CreateComment(ctx, "stackrox", "stackrox", prNumber, &retestComment)
				handleError(err)
				log.Printf("#%d commented: %s", prNumber, comment.GetURL())
			}
		}
	}
}

type Status struct {
	Context string `json:"context"`
	State   string `json:"state"`
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
