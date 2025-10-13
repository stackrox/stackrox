package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/go-github/v61/github"
)

var (
	headSHA           = flag.String("head-SHA", "", "the SHA for the workflow run")
	workflowFileName  = flag.String("workflow", "build.yaml", "the workflow of interest")
	generateProwError = flag.Bool("prow-error", true, "generate output that will stand out in prow")
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

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

	conclusion := lastRun.GetConclusion()
	if *generateProwError && conclusion != "" && conclusion != "success" {
		fmt.Printf("ERROR: GitHub Actions workflow %s did not complete successfully\n", *workflowFileName)
	}

	fmt.Printf("status: %v\nconclusion: %v\nURL: %v\n",
		lastRun.GetStatus(), conclusion, lastRun.GetHTMLURL())
}
