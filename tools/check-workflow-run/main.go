package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v61/github"
	flag "github.com/spf13/pflag"
)

var (
	headSHA          = flag.String("head-SHA", "", "the SHA for the workflow run")
	workflowFileName = flag.String("workflow", "build.yaml", "the workflow of interest")
)

// lookup useful information for a github action workflows most recent run
// relating to a SHA.
func main() {
	flag.Parse()

	if *headSHA == "" {
		fmt.Println("--head-SHA is required")
		os.Exit(2)
	}

	client := github.NewClient(nil)
	ctx := context.Background()

	runs, _, err := client.Actions.ListWorkflowRunsByFileName(
		ctx, "stackrox", "stackrox", *workflowFileName,
		&github.ListWorkflowRunsOptions{
			HeadSHA: *headSHA,
		},
	)
	if err != nil {
		fmt.Println("Cannot list workflow runs", err)
		os.Exit(2)
	}

	if *runs.TotalCount == 0 {
		fmt.Println("no runs")
		os.Exit(1)
	}

	lastRun := runs.WorkflowRuns[*runs.TotalCount-1]

	fmt.Printf("status: %v\nconclusion: %v\nURL: %v\n",
		lastRun.GetStatus(), lastRun.GetConclusion(), lastRun.GetHTMLURL())
}
