//go:build sql_integration

package tests

import (
	"context"
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestProfileIngestion(t *testing.T) {
	c := NewCentral(t)
	c.Start()

	sensor := NewFakeSensor(t, "test-cluster")
	sensor.ConnectTo(c)

	// Inject a compliance profile via the sensor stream.
	sensor.Inject(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
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
			},
		},
	})

	// Query via the compliance profile API.
	apiConn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(bufconnDialer(c.listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer apiConn.Close()

	profileClient := v2.NewComplianceProfileServiceClient(apiConn)

	var resp *v2.ListComplianceProfilesResponse
	assert.Eventually(t, func() bool {
		resp, err = profileClient.ListComplianceProfiles(context.Background(),
			&v2.ProfilesForClusterRequest{ClusterId: sensor.ClusterID()})
		return err == nil && len(resp.GetProfiles()) == 1
	}, 5*time.Second, 100*time.Millisecond)

	require.NoError(t, err)
	require.Len(t, resp.GetProfiles(), 1)
	assert.Equal(t, "ocp4-cis", resp.GetProfiles()[0].GetName())
	assert.Equal(t, "CIS OpenShift Benchmark", resp.GetProfiles()[0].GetTitle())
}
