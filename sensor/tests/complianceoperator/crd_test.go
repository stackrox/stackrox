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

func Test_ComplianceOperatorV2CRDsDetection(t *testing.T) {
	t.Setenv(env.ConnectionRetryInitialInterval.EnvVar(), "1s")
	t.Setenv(env.ConnectionRetryMaxInterval.EnvVar(), "2s")

	c, err := helper.NewContextWithConfig(t, helper.Config{
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})
	t.Cleanup(c.Stop)

	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		testContext.WaitForSyncEvent(t, 10*time.Second)
		testContext.GetFakeCentral().ClearReceivedBuffer()
		// Create CO CRDs
		deleteFn, err := testContext.ApplyWithManifestDir(context.Background(), "../../../tests/complianceoperator/crds", "*")
		require.NoError(t, err)
		defer func() {
			utils.IgnoreError(deleteFn)
		}()
		testContext.WaitForSyncEvent(t, 10*time.Second)
	}))

}
