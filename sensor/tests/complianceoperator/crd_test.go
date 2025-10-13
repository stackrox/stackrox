package complianceoperator

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

func Test_ComplianceOperatorCRDsDetection(t *testing.T) {
	t.Setenv(env.ConnectionRetryInitialInterval.EnvVar(), "1s")
	t.Setenv(env.ConnectionRetryMaxInterval.EnvVar(), "2s")

	c, err := helper.NewContextWithConfig(t, helper.Config{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})
	t.Cleanup(c.Stop)

	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		// Wait for the first sync
		testContext.WaitForSyncEvent(t, 10*time.Second)
		testContext.GetFakeCentral().ClearReceivedBuffer()

		// Create the Compliance Operator CRDs
		deleteCRDsFn, err := testContext.ApplyWithManifestDir(context.Background(), "../../../tests/complianceoperator/crds", "*")
		require.NoError(t, err)

		// Just in case the tests fails we don't want to leave resources behind
		t.Cleanup(func() {
			utils.IgnoreError(deleteCRDsFn)
		})

		// Sensor should sync again after the CRDs are detected
		testContext.WaitForSyncEventf(t, 10*time.Second, "expected restart connection after CRDs are detected")
		testContext.GetFakeCentral().ClearReceivedBuffer()

		// Delete the Compliance Operator CRDs
		require.NoError(t, deleteCRDsFn())

		// Sensor should sync again after the CRDs removal is detected
		testContext.WaitForSyncEventf(t, 10*time.Second, "expected restart connection after CRDs are removed")
	}))

}
