package connection

import (
	"testing"
	"time"

	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

func Test_ConnectionBroke(t *testing.T) {
	c, err := helper.NewContext(t)
	require.NoError(t, err)

	c.RunTest(helper.WithTestCase(func(t *testing.T, testContext *helper.TestContext, _ map[string]k8s.Object) {
		fakeCentral := testContext.GetFakeCentral()

		// Force gRPC server to stop
		fakeCentral.ServerPointer.Stop()

		// Give sometime to sensor realize that the connection is broken
		time.Sleep(time.Second)

		assert.False(t, testContext.SensorStopped())
	}))

}
