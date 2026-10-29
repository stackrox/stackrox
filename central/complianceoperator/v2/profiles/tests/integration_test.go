//go:build sql_integration

package tests

import (
	"context"
	"testing"
	"time"

	framework "github.com/stackrox/rox/central/tests"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileIngestion(t *testing.T) {
	c := framework.NewCentral(t)
	c.Start()

	sensor := framework.NewFakeSensor(t, "test-cluster")
	sensor.ConnectTo(c)

	sensor.Inject(framework.NewSensorEvent(&central.SensorEvent{
		Id:     "profile-uid-1",
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
			ComplianceOperatorProfileV2: &central.ComplianceOperatorProfileV2{
				Id:             "profile-uid-1",
				ProfileId:      "xccdf_org.ssgproject.content_profile_cis",
				Name:           "ocp4-cis",
				ProfileVersion: "0.1.0",
				Title:          "CIS OpenShift Benchmark",
				Description:    "CIS benchmark for OpenShift",
			},
		},
	}))

	sensorTwo := framework.NewFakeSensor(t, "test-cluster-2")
	sensorTwo.ConnectTo(c)
	sensorTwo.Inject(framework.NewSensorEvent(&central.SensorEvent{
		Id:     "profile-uid-2",
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
			ComplianceOperatorProfileV2: &central.ComplianceOperatorProfileV2{
				Id:             "profile-uid-2",
				ProfileId:      "xccdf_org.ssgproject.content_profile_cis",
				Name:           "ocp4-cis",
				ProfileVersion: "0.1.0",
				Title:          "CIS OpenShift Benchmark",
				Description:    "CIS benchmark for OpenShift",
			},
		},
	}))

	profileClient := v2.NewComplianceProfileServiceClient(c.APIConn())

	var resp *v2.ListComplianceProfilesResponse
	var err error
	assert.Eventually(t, func() bool {
		resp, err = profileClient.ListComplianceProfiles(context.Background(),
			&v2.ProfilesForClusterRequest{ClusterId: sensor.ClusterID()})
		return err == nil && len(resp.GetProfiles()) == 1
	}, 5*time.Second, 100*time.Millisecond)

	require.NoError(t, err)
	require.Len(t, resp.GetProfiles(), 1)
	assert.Equal(t, "ocp4-cis", resp.GetProfiles()[0].GetName())
	assert.Equal(t, "CIS OpenShift Benchmark", resp.GetProfiles()[0].GetTitle())

	assert.Eventually(t, func() bool {
		resp, err = profileClient.ListComplianceProfiles(context.Background(),
			&v2.ProfilesForClusterRequest{ClusterId: sensorTwo.ClusterID()})
		return err == nil && len(resp.GetProfiles()) == 1
	}, 5*time.Second, 100*time.Millisecond)

	require.NoError(t, err)
	require.Len(t, resp.GetProfiles(), 1)
	assert.Equal(t, "ocp4-cis", resp.GetProfiles()[0].GetName())
	assert.Equal(t, "CIS OpenShift Benchmark", resp.GetProfiles()[0].GetTitle())
}
