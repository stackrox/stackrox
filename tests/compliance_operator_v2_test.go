//go:build compliance

package tests

import (
	"context"
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
)

// ACS API test suite for integration testing for the Compliance Operator.
func TestComplianceV2Integration(t *testing.T) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceIntegrationServiceClient(conn)

	q := &v2.RawQuery{Query: ""}
	resp, err := client.ListComplianceIntegrations(context.TODO(), q)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, resp.Integrations, 1, "failed to assert there is only a single compliance integration")
	assert.Equal(t, resp.Integrations[0].ClusterName, "remote", "failed to find integration for cluster called \"remote\"")
	assert.Equal(t, resp.Integrations[0].Namespace, "openshift-compliance", "failed to find integration for \"openshift-compliance\" namespace")
}

func TestComplianceV2ProfileCount(t *testing.T) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)
	profileCount, err := client.GetComplianceProfileCount(context.TODO(), &v2.RawQuery{Query: ""})
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, profileCount.Count, int32(0), "unable to verify any compliance profiles were ingested")
}

func TestComplianceV2ProfileGet(t *testing.T) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)
	profile, err := client.GetComplianceProfile(context.TODO(), &v2.ResourceByID{Id: "ocp4-cis"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(profile.Rules), 0, "failed to verify ocp4-cis profile contains any rules")
}
