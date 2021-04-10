package connection

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stretchr/testify/assert"
)

// TestGetPolicySyncMsgFromPolicies verifies that the sensor connection is
// capable of downgrading policies to the version known of the underlying
// sensor. The test uses specific policy versions and not a general approach.
func TestGetPolicySyncMsgFromPolicies(t *testing.T) {
	centralVersion := policyversion.CurrentVersion()
	sensorVersion := policyversion.Version1()
	sensorHello := &central.SensorHello{
		PolicyVersion: sensorVersion.String(),
	}
	sensorMockConn := &sensorConnection{
		sensorHello: sensorHello,
	}
	policy := &storage.Policy{
		PolicyVersion: centralVersion.String(),
	}

	msg, err := sensorMockConn.getPolicySyncMsgFromPolicies([]*storage.Policy{policy})
	assert.NoError(t, err)

	policySync := msg.GetPolicySync()
	assert.NotNil(t, policySync)
	assert.NotEmpty(t, policySync.Policies)
	assert.Equal(t, sensorVersion.String(), policySync.Policies[0].GetPolicyVersion())
}
