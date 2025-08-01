//go:build test_e2e

package tests

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	coNamespace = "openshift-compliance"

	// rhcos4 constants
	rhcosProfileName    = "rhcos4-moderate"
	masterMachineConfig = "rhcos4-moderate-master"
	workerMachineConfig = "rhcos4-moderate-worker"

	chmodControl = "rhcos4-moderate:audit-rules-dac-modification-chmod"
	chmodRule    = "rhcos4-audit-rules-dac-modification-chmod"
	chownControl = "rhcos4-moderate:audit-rules-dac-modification-chown"
	chownRule    = "rhcos4-audit-rules-dac-modification-chown"
	uidControl   = "rhcos4-moderate:accounts-no-uid-except-zero"

	// ocp4 constants
	ocp4ProfileName        = "ocp4-moderate"
	envVarControl          = "ocp4-moderate:secrets-no-environment-variables"
	envVarRule             = "ocp4-secrets-no-environment-variables"
	externalStorageControl = "ocp4-moderate:secrets-consider-external-storage"

	unusedProfile = "rhcos4-anssi-bp28-high"
)

const defaultWaitTime = 30 * time.Second
const defaultSleepTime = 5 * time.Second
const defaultTickTime = 2 * time.Second

type collectT struct {
	t *testing.T
	c *assert.CollectT
}

func (c *collectT) Fatalf(format string, args ...interface{}) {
	if c.t != nil {
		c.t.Fatalf(format, args...)
	}
}

func (c *collectT) Errorf(format string, args ...interface{}) {
	if c.c != nil {
		c.Errorf(format, args...)
	}
}

func (c *collectT) FailNow() {
	if c.c != nil {
		c.FailNow()
	}
}

func wrapCollectT(t *testing.T, c *assert.CollectT) *collectT {
	return &collectT{
		t: t,
		c: c,
	}
}

func getCurrentComplianceResults(t testutils.T) (rhcos, ocp *storage.ComplianceRunResults) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
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
		assert.NotEqual(t, unusedProfile, run.GetStandardId())
		switch run.GetStandardId() {
		case rhcosProfileName:
			rhcosRun = run
		case ocp4ProfileName:
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

	rhcosResults, _ := complianceService.GetRunResults(context.Background(), &v1.GetComplianceRunResultsRequest{
		StandardId: rhcosRun.GetStandardId(),
		ClusterId:  rhcosRun.GetClusterId(),
	})

	// ocp4 results
	ocpResults, _ := complianceService.GetRunResults(context.Background(), &v1.GetComplianceRunResultsRequest{
		StandardId: ocpRun.GetStandardId(),
		ClusterId:  ocpRun.GetClusterId(),
	})

	return rhcosResults.GetResults(), ocpResults.GetResults()
}

func checkResult(t assert.TestingT, results map[string]*storage.ComplianceResultValue, rule string, state storage.ComplianceState) {
	assert.Equal(t, state.String(), results[rule].GetOverallState().String(), "expected result states did not match")
}

func checkMachineConfigResult(t assert.TestingT, entityResults map[string]*storage.ComplianceRunResults_EntityResults, machineConfig, rule string, state storage.ComplianceState) {
	checkResult(t, entityResults[machineConfig].GetControlResults(), rule, state)
}

func checkBaseResults(t *testing.T) {
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		rhcosResults, ocpResults := getCurrentComplianceResults(wrapCollectT(t, c))
		require.NotNil(c, rhcosResults)
		require.NotNil(c, ocpResults)

		machineConfigResults := rhcosResults.GetMachineConfigResults()
		checkMachineConfigResult(c, machineConfigResults, masterMachineConfig, chmodControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
		checkMachineConfigResult(c, machineConfigResults, masterMachineConfig, chownControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
		checkMachineConfigResult(c, machineConfigResults, masterMachineConfig, uidControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)

		checkMachineConfigResult(c, machineConfigResults, workerMachineConfig, chmodControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
		checkMachineConfigResult(c, machineConfigResults, workerMachineConfig, chownControl, storage.ComplianceState_COMPLIANCE_STATE_SKIP)
		checkMachineConfigResult(c, machineConfigResults, workerMachineConfig, uidControl, storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)

		clusterResults := ocpResults.GetClusterResults().GetControlResults()
		checkResult(c, clusterResults, envVarControl, storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)
		checkResult(c, clusterResults, externalStorageControl, storage.ComplianceState_COMPLIANCE_STATE_SKIP)
	}, defaultWaitTime, defaultTickTime)
}

func TestComplianceOperatorResults(t *testing.T) {
	// Base case happy path, existing compliance operator data
	checkBaseResults(t)
}

func getDynamicClientGenerator(t *testing.T) dynamic.Interface {
	kubeconfig := os.Getenv("KUBECONFIG")
	if len(kubeconfig) == 0 {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube/config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err)

	dynamicClientGenerator, err := dynamic.NewForConfig(config)
	require.NoError(t, err)
	return dynamicClientGenerator
}

func TestDeleteAndAddRule(t *testing.T) {
	checkBaseResults(t)

	dynamicClientGenerator := getDynamicClientGenerator(t)
	// Remove a rule from the profile and verify it's gone from the results
	ruleClient := dynamicClientGenerator.Resource(complianceoperator.Rule.GroupVersionResource()).Namespace(coNamespace)
	rule, err := ruleClient.Get(context.Background(), envVarRule, metav1.GetOptions{})
	assert.NoError(t, err)

	err = ruleClient.Delete(context.Background(), envVarRule, metav1.DeleteOptions{})
	assert.NoError(t, err)

	time.Sleep(defaultSleepTime)

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		rhcosResults, ocpResults := getCurrentComplianceResults(wrapCollectT(t, c))
		require.NotNil(c, rhcosResults)
		require.NotNil(c, ocpResults)

		machineConfigResults := rhcosResults.GetMachineConfigResults()
		checkMachineConfigResult(c, machineConfigResults, masterMachineConfig, chmodControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
		checkMachineConfigResult(c, machineConfigResults, masterMachineConfig, chownControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
		checkMachineConfigResult(c, machineConfigResults, masterMachineConfig, uidControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)

		checkMachineConfigResult(c, machineConfigResults, workerMachineConfig, chmodControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
		checkMachineConfigResult(c, machineConfigResults, workerMachineConfig, chownControl, storage.ComplianceState_COMPLIANCE_STATE_SKIP)
		checkMachineConfigResult(c, machineConfigResults, workerMachineConfig, uidControl, storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)

		clusterResults := ocpResults.GetClusterResults().GetControlResults()
		checkResult(c, clusterResults, externalStorageControl, storage.ComplianceState_COMPLIANCE_STATE_SKIP)
		assert.Nil(c, clusterResults[envVarControl])
	}, defaultWaitTime, 2*time.Second)

	rule.SetResourceVersion("")
	_, err = ruleClient.Create(context.Background(), rule, metav1.CreateOptions{})
	assert.NoError(t, err)

	time.Sleep(defaultSleepTime)

	checkBaseResults(t)
}

func TestDeleteAndAddScanSettingBinding(t *testing.T) {
	checkBaseResults(t)

	dynamicClientGenerator := getDynamicClientGenerator(t)

	// Delete a scansettingbinding
	ssbClient := dynamicClientGenerator.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Namespace(coNamespace)
	ssb, err := ssbClient.Get(context.Background(), rhcosProfileName, metav1.GetOptions{})
	assert.NoError(t, err)

	err = ssbClient.Delete(context.Background(), rhcosProfileName, metav1.DeleteOptions{})
	assert.NoError(t, err)

	time.Sleep(defaultSleepTime)

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		rhcosResults, ocpResults := getCurrentComplianceResults(wrapCollectT(t, c))
		assert.Nil(c, rhcosResults)
		require.NotNil(c, ocpResults)

		clusterResults := ocpResults.GetClusterResults().GetControlResults()
		checkResult(c, clusterResults, envVarControl, storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)
		checkResult(c, clusterResults, externalStorageControl, storage.ComplianceState_COMPLIANCE_STATE_SKIP)
	}, defaultWaitTime, defaultTickTime)

	ssb.SetResourceVersion("")
	_, err = ssbClient.Create(context.Background(), ssb, metav1.CreateOptions{})
	assert.NoError(t, err)

	time.Sleep(defaultSleepTime)

	checkBaseResults(t)
}

func TestDeleteAndAddProfile(t *testing.T) {
	checkBaseResults(t)

	dynamicClientGenerator := getDynamicClientGenerator(t)

	// Remove a profile and verify that the profile is gone
	profileClient := dynamicClientGenerator.Resource(complianceoperator.Profile.GroupVersionResource()).Namespace(coNamespace)
	profile, err := profileClient.Get(context.Background(), rhcosProfileName, metav1.GetOptions{})
	assert.NoError(t, err)

	err = profileClient.Delete(context.Background(), rhcosProfileName, metav1.DeleteOptions{})
	require.NoError(t, err)

	time.Sleep(defaultSleepTime)

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		rhcosResults, ocpResults := getCurrentComplianceResults(wrapCollectT(t, c))
		assert.Nil(c, rhcosResults)
		require.NotNil(c, ocpResults)

		clusterResults := ocpResults.GetClusterResults().GetControlResults()
		checkResult(c, clusterResults, envVarControl, storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)
		checkResult(c, clusterResults, externalStorageControl, storage.ComplianceState_COMPLIANCE_STATE_SKIP)
	}, defaultWaitTime, defaultTickTime)

	profile.SetResourceVersion("")
	_, err = profileClient.Create(context.Background(), profile, metav1.CreateOptions{})
	assert.NoError(t, err)

	time.Sleep(defaultSleepTime)

	checkBaseResults(t)
}

func TestUpdateProfile(t *testing.T) {
	checkBaseResults(t)

	dynamicClientGenerator := getDynamicClientGenerator(t)

	// Remove a profile and verify that the profile is gone
	profileClient := dynamicClientGenerator.Resource(complianceoperator.Profile.GroupVersionResource()).Namespace(coNamespace)
	profileObj, err := profileClient.Get(context.Background(), rhcosProfileName, metav1.GetOptions{})
	assert.NoError(t, err)

	originalRules := profileObj.Object["rules"]

	profileObj.Object["rules"] = []string{
		chmodRule,
		chownRule,
	}
	profileObj, err = profileClient.Update(context.Background(), profileObj, metav1.UpdateOptions{})
	assert.NoError(t, err)

	time.Sleep(defaultSleepTime)

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		rhcosResults, ocpResults := getCurrentComplianceResults(wrapCollectT(t, c))
		require.NotNil(c, rhcosResults)
		require.NotNil(c, ocpResults)

		machineConfigResults := rhcosResults.GetMachineConfigResults()
		checkMachineConfigResult(c, machineConfigResults, masterMachineConfig, chmodControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
		checkMachineConfigResult(c, machineConfigResults, masterMachineConfig, chownControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)

		checkMachineConfigResult(c, machineConfigResults, workerMachineConfig, chmodControl, storage.ComplianceState_COMPLIANCE_STATE_FAILURE)
		checkMachineConfigResult(c, machineConfigResults, workerMachineConfig, chownControl, storage.ComplianceState_COMPLIANCE_STATE_SKIP)

		clusterResults := ocpResults.GetClusterResults().GetControlResults()
		checkResult(c, clusterResults, envVarControl, storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)
		checkResult(c, clusterResults, externalStorageControl, storage.ComplianceState_COMPLIANCE_STATE_SKIP)
	}, defaultWaitTime, defaultTickTime)

	profileObj.Object["rules"] = originalRules
	_, err = profileClient.Update(context.Background(), profileObj, metav1.UpdateOptions{})
	assert.NoError(t, err)

	time.Sleep(defaultSleepTime)

	checkBaseResults(t)
}
