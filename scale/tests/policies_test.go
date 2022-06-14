package tests

import (
	"testing"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/defaults/policies"
	"github.com/stackrox/stackrox/pkg/roxctl/common"
)

func submitDryRunJob(service v1.PolicyServiceClient, policy *storage.Policy, jobChan chan string) func() error {
	return func() error {
		res, err := service.SubmitDryRunPolicyJob(common.Context(), policy)
		if err == nil {
			jobChan <- res.JobId
		}

		return err
	}
}

func queryJobTillCompletion(service v1.PolicyServiceClient, jobID string) func() error {
	return func() (err error) {
		for {
			res, err := service.QueryDryRunJobStatus(common.Context(), &v1.JobId{
				JobId: jobID,
			})

			if err != nil {
				return err
			}

			if !res.Pending {
				break
			}
		}

		return nil
	}
}

func collectAndQuery(service v1.PolicyServiceClient, jobChan chan string) func() error {
	return func() error {
		wg := concurrency.NewWaitGroup(0)
		for j := range jobChan {
			asyncWithWaitGroup(queryJobTillCompletion(service, j), &wg)
		}

		<-wg.Done()
		return nil
	}
}

func submitJobs(service v1.PolicyServiceClient, jobChan chan string, policies []*storage.Policy) func() error {
	return func() error {
		wg := concurrency.NewWaitGroup(0)
		for idx := 0; idx < len(policies); idx++ {
			asyncWithWaitGroup(submitDryRunJob(service, policies[idx], jobChan), &wg)
		}

		<-wg.Done()
		close(jobChan)
		return nil
	}
}

func BenchmarkDryRunPolicies(b *testing.B) {
	envVars := getEnvVars()

	connection, err := getConnection(envVars.endpoint, envVars.password)
	if err != nil {
		log.Fatal(err)
	}

	policyService := v1.NewPolicyServiceClient(connection)
	defPolicies, err := policies.DefaultPolicies()
	if err != nil {
		log.Fatal(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		jobChan := make(chan string, len(defPolicies))
		wg := concurrency.NewWaitGroup(0)
		// Consumer of submitted jobs.
		asyncWithWaitGroup(collectAndQuery(policyService, jobChan), &wg)
		// Producer of dry run policy jobs.
		asyncWithWaitGroup(submitJobs(policyService, jobChan, defPolicies), &wg)
		<-wg.Done()
	}
}
