package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	prowUrl        = "https://gangway-ci.apps.ci.l2s4.p1.openshiftapps.com/v1/executions"
	jobUrlTemplate = "https://prow.ci.openshift.org/job-history/gs/test-platform-results-public/logs/"
)

type RerunProwPostsubmitRequest struct {
	JobName          string `json:"job_name"`
	JobExecutionType string `json:"job_execution_type"`
	Refs             Refs   `json:"refs"`
}

type Refs struct {
	Org     string `json:"org"`
	Repo    string `json:"repo"`
	BaseRef string `json:"base_ref"`
	BaseSha string `json:"base_sha"`
}

type JobExecutionType int

const (
	JobExecutionTypePeriodic JobExecutionType = iota + 1
	JobExecutionTypePostsubmit
)

func (j JobExecutionType) String() string {
	return strconv.Itoa(int(j))
}

func buildRequestBody(jobName string, jobExecutionType JobExecutionType, org string, repo string, baseRef string, baseSha string) []byte {
	body := RerunProwPostsubmitRequest{
		JobName:          jobName,
		JobExecutionType: jobExecutionType.String(),
		Refs: Refs{
			Org:     org,
			Repo:    repo,
			BaseRef: baseRef,
			BaseSha: baseSha,
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		log.Fatalf("Failed to marshal request: %v", err)
	}
	return data
}

func getToken() string {
	token := os.Getenv("PROW_TOKEN")
	if token == "" {
		log.Fatalf("PROW_TOKEN is not set. Set it with `export PROW_TOKEN=\"$(oc whoami -t)\"`")
	}
	return token
}

func postRequest(requestBody []byte, token string) (int, string) {
	req, err := http.NewRequest("POST", prowUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Failed to post request: %v", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
	}
	bodyString := string(body)

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to post request: %v. Response body: %s", resp.Status, bodyString)
	}
	return resp.StatusCode, bodyString
}

func main() {
	token := getToken()

	var jobName string
	var org string
	var repo string
	var baseRef string
	var baseSha string

	flag.StringVar(&jobName, "job-name", "", "The name of the job to re-run")
	flag.StringVar(&org, "org", "stackrox", "The org of the job to re-run")
	flag.StringVar(&repo, "repo", "stackrox", "The repo of the job to re-run")
	flag.StringVar(&baseRef, "base-ref", "", "The base ref of the job to re-run")
	flag.StringVar(&baseSha, "base-sha", "", "The base sha of the job to re-run")
	flag.Parse()

	if jobName == "" || baseRef == "" || baseSha == "" {
		flag.PrintDefaults()
		log.Fatalf("Missing required flags")
	}

	log.Printf("Retrigging job: %s (%s/%s @ %s (%s))", jobName, org, repo, baseSha, baseRef)

	requestBody := buildRequestBody(jobName, JobExecutionTypePostsubmit, org, repo, baseRef, baseSha)
	postRequest(requestBody, token)
	jobUrl := fmt.Sprintf("%s/%s", jobUrlTemplate, jobName)
	log.Printf("Successfully retriggered job: %s\n", jobUrl)
}
