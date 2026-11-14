package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	appCIServerHost = "ci.l2s4.p1.openshiftapps.com"
	appCIServer     = "https://api." + appCIServerHost + ":6443"
	prowUrl         = "https://gangway-ci.apps." + appCIServerHost + "/v1/executions"
	jobUrlTemplate  = "https://prow.ci.openshift.org/job-history/gs/test-platform-results-public/logs"
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

func ocRun(args ...string) error {
	cmd := exec.Command("oc", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ocOutput(args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("oc", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%w\nstdout: %s\nstderr: %s", err,
			strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func login() {
	fmt.Printf("This will run: oc login --server=%s --web\n", appCIServer)
	fmt.Print("A browser will open for you to complete the login. Continue? [y/n] ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "" && answer != "y" && answer != "yes" {
		log.Fatalf("Login cancelled by user")
	}
	if err := ocRun("login", "--server="+appCIServer, "--web"); err != nil {
		log.Fatalf("Failed to login: %v", err)
	}
}

func getToken() string {
	token, err := ocOutput("whoami", "-t")
	if err != nil {
		log.Fatalf("Failed to get token via `oc whoami -t`: %v", err)
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

	log.Printf("Response body: %s", bodyString)
	log.Printf("Response status: %s", resp.Status)

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to post request: %v. Response body: %s", resp.Status, bodyString)
	}
	return resp.StatusCode, bodyString
}

func main() {
	var jobName string
	var org string
	var repo string
	var baseRef string
	var baseSha string

	flag.StringVar(&jobName, "job-name", "", "The name of the job to re-run")
	flag.StringVar(&org, "org", "stackrox", "The org of the job to re-run")
	flag.StringVar(&repo, "repo", "stackrox", "The repo of the job to re-run")
	flag.StringVar(&baseRef, "base-ref", "", "The base ref of the job to re-run")
	flag.StringVar(&baseSha, "base-sha", "", "The base sha of the job to re-run (40 characters long)")
	flag.Parse()

	if jobName == "" || baseRef == "" || baseSha == "" {
		flag.PrintDefaults()
		log.Fatalf("Missing required flags")
	}

	if len(baseSha) != 40 {
		log.Fatalf("Base SHA must be 40 characters long")
	}

	login()
	token := getToken()

	log.Printf("Retrigging job: %s (%s/%s @ %s (%s))", jobName, org, repo, baseSha, baseRef)

	requestBody := buildRequestBody(jobName, JobExecutionTypePostsubmit, org, repo, baseRef, baseSha)
	postRequest(requestBody, token)
	jobUrl := fmt.Sprintf("%s/%s", jobUrlTemplate, jobName)
	log.Printf("Successfully retriggered job: %s\n", jobUrl)
}
