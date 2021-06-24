package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// rhcos4 constants
	masterMachineConfig = "rhcos4-moderate-master"
	workerMachineConfig = "rhcos4-moderate-worker"

	chmodRule = "rhcos4-moderate:audit-rules-dac-modification-chmod"
	chownRule = "rhcos4-moderate:audit-rules-dac-modification-chown"
	uidRule   = "rhcos4-moderate:accounts-no-uid-except-zero"

	// ocp4 constants
	envVarRule      = "ocp4-moderate:secrets-no-environment-variables"
	externalStorage = "ocp4-moderate:secrets-consider-external-storage"
)

func getCurrentComplianceResults(t *testing.T) (rhcos, ocp *storage.ComplianceRunResults) {
	conn := testutils.GRPCConnectionToCentral(t)
	managementService := v1.NewComplianceManagementServiceClient(conn)

	resp, err := managementService.TriggerRuns(context.Background(), &v1.TriggerComplianceRunsRequest{
		Selection: &v1.ComplianceRunSelection{
			ClusterId:  "*",
			StandardId: "*",
		},
	})
	require.NoError(t, err)

	var rhcosRun, ocpRun *v1.ComplianceRun
	for _, run := range resp.StartedRuns {
		// Ensure the profile not referenced by a scan setting binding is not run
		assert.NotEqual(t, "rhcos4-anssi-bp28-high", run.GetStandardId())
		switch run.GetStandardId() {
		case "rhcos4-moderate":
			rhcosRun = run
		case "ocp4-moderate":
			ocpRun = run
		}
	}

	// Retry logic
	// Wait for the run to complete
	err = retry.WithRetry(func() error {
		statusRunResp, err := managementService.GetRunStatuses(context.Background(), &v1.GetComplianceRunStatusesRequest{
			RunIds: []string{rhcosRun.GetId(), ocpRun.GetId()},
		})
		require.NoError(t, err)
		assert.Empty(t, statusRunResp.GetInvalidRunIds())
		assert.NotEmpty(t, statusRunResp.GetRuns())

		finished := true
		for _, run := range statusRunResp.GetRuns() {
			if run.GetState() != v1.ComplianceRun_FINISHED {
				finished = false
				log.Infof("Run for %v is in state %v", run.GetStandardId(), run.GetState())
			}
		}
		if finished {
			return nil
		}
		return errors.New("not all runs are finished")
	}, retry.BetweenAttempts(func(previousAttemptNumber int) {
		time.Sleep(2 * time.Second)
	}), retry.Tries(10))
	assert.NoError(t, err)

	complianceService := v1.NewComplianceServiceClient(conn)

	// rhcos4 results

	rhcosResults, err := complianceService.GetRunResults(context.Background(), &v1.GetComplianceRunResultsRequest{
		StandardId: rhcosRun.GetStandardId(),
		ClusterId:  rhcosRun.GetClusterId(),
	})
	assert.NoError(t, err)

	// ocp4 results
	ocpResults, err := complianceService.GetRunResults(context.Background(), &v1.GetComplianceRunResultsRequest{
		StandardId: ocpRun.GetStandardId(),
		ClusterId:  ocpRun.GetClusterId(),
	})
	assert.NoError(t, err)

	return rhcosResults.GetResults(), ocpResults.GetResults()
}

func checkResult(t *testing.T, results map[string]*storage.ComplianceResultValue, rule string, state storage.ComplianceState) {
	assert.Equal(t, state, results[rule].GetOverallState())
}

func checkMachineConfigResult(t *testing.T, entityResults map[string]*storage.ComplianceRunResults_EntityResults, machineConfig, rule string, state storage.ComplianceState) {
	checkResult(t, entityResults[machineConfig].GetControlResults(), rule, state)
}

func TestComplianceOperatorResults(t *testing.T) {
	rhcosResults, ocpResults := getCurrentComplianceResults(t)
	require.NotNil(t, rhcosResults)
	require.NotNil(t, ocpResults)

	// Base case happy path, existing compliance operator data
	machineConfigResults := rhcosResults.GetMachineConfigResults()
	checkMachineConfigResult(t, machineConfigResults, masterMachineConfig, chmodRule, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
	checkMachineConfigResult(t, machineConfigResults, masterMachineConfig, chownRule, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
	checkMachineConfigResult(t, machineConfigResults, masterMachineConfig, uidRule, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)

	checkMachineConfigResult(t, machineConfigResults, workerMachineConfig, chmodRule, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
	checkMachineConfigResult(t, machineConfigResults, workerMachineConfig, chownRule, storage.ComplianceState_COMPLIANCE_STATE_SKIP)
	checkMachineConfigResult(t, machineConfigResults, workerMachineConfig, uidRule, storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)

	clusterResults := ocpResults.GetClusterResults().GetControlResults()
	checkResult(t, clusterResults, envVarRule, storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)
	checkResult(t, clusterResults, externalStorage, storage.ComplianceState_COMPLIANCE_STATE_SKIP)
}
