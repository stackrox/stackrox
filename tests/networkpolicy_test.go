package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	netpolDeploymentName = "nginx-deployment"
)

func getFakePolicyRequest(policyName, fieldName string) *v1.PostPolicyRequest {
	return &v1.PostPolicyRequest{
		Policy: &storage.Policy{
			Id: "",
			LifecycleStages: []storage.LifecycleStage{
				storage.LifecycleStage_DEPLOY,
			},
			Name:               policyName,
			IsDefault:          false,
			CriteriaLocked:     false,
			MitreVectorsLocked: false,
			Severity:           storage.Severity_LOW_SEVERITY,
			Categories:         []string{"Anomalous Activity"},
			PolicySections: []*storage.PolicySection{
				{
					SectionName: "example",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       fieldName,
							BooleanOperator: 0,
							Negate:          false,
							Values: []*storage.PolicyValue{
								{
									Value: "true",
								},
							},
						},
					},
				},
			},
		},
		EnableStrictValidation: false,
	}
}

func createPolicyIfMissing(t *testing.T, client v1.PolicyServiceClient, req *v1.PostPolicyRequest) (func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	p, err := client.PostPolicy(ctx, req)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.Internal && strings.Contains(s.Message(), "Could not add policy due to name validation") {
				return func() {}, nil
			}
		}
		return nil, err
	}
	return func() {
		deletePolicy(t, client, p.Id)
	}, nil
}

func deletePolicy(t *testing.T, client v1.PolicyServiceClient, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := client.DeletePolicy(ctx, &v1.ResourceByID{Id: id})
	assert.NoError(t, err)
}

func applyDeployment(t *testing.T) func() {
	// Create deployment that doesn't have any network policies
	setupDeploymentFromFile(t, netpolDeploymentName, "yamls/nginx.yaml")
	return func() {
		teardownDeploymentFromFile(t, netpolDeploymentName, "yamls/nginx.yaml")
	}
}

func getAlertsForPolicy(t *testing.T, client v1.AlertServiceClient) []*storage.ListAlert {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var alerts []*storage.ListAlert
	resp, err := client.ListAlerts(ctx, &v1.ListAlertsRequest{
		Query: "Policy:Automated Missing Ingress Policy+Namespace:default",
	})

	if err != nil {
		t.Errorf("Failed to get alerts: %s", err)
		t.FailNow()
	}

	alerts = resp.GetAlerts()
	return alerts
}

func applyNetworkPolicy(t *testing.T) func() {
	// Create deployment that doesn't have any network policies
	applyFile(t, "yamls/allow-ingress-netpol.yaml")
	return func() {
		teardownFile(t, "yamls/allow-ingress-netpol.yaml")
	}
}

func CheckIfCentralHasFeatureFlag(t *testing.T, conn *grpc.ClientConn) bool {
	ffClient := v1.NewFeatureFlagServiceClient(conn)

	flags, err := ffClient.GetFeatureFlags(context.Background(), &v1.Empty{})
	assert.NoError(t, err)

	for _, flag := range flags.FeatureFlags {
		if flag.Name == features.NetworkPolicySystemPolicy.Name() {
			return flag.Enabled
		}
	}

	return false
}

func Test_GetViolationForIngressPolicy(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)
	if !CheckIfCentralHasFeatureFlag(t, conn) {
		t.Skip("Feature flag disabled")
	}

	testCases := map[string]struct {
		policyName        string
		policyField       string
		networkPolicyFile string
	}{
		"Should have ingress": {
			policyName:        "[Automated] Should have ingress policy",
			policyField:       "Missing Ingress Network Policy",
			networkPolicyFile: "allow-ingress-netpol.yaml",
		},
		"Should have egress": {
			policyName:        "[Automated] Should have egress policy",
			policyField:       "Missing Egress Network Policy",
			networkPolicyFile: "allow-egress-netpol.yaml",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			policyService := v1.NewPolicyServiceClient(conn)
			alertService := v1.NewAlertServiceClient(conn)

			// Policy creation
			deletePolicyF, err := createPolicyIfMissing(t, policyService, getFakePolicyRequest(testCase.policyName, testCase.policyField))
			defer deletePolicyF()
			assert.NoError(t, err)

			// Deployment creation
			deleteDeploymentF := applyDeployment(t)
			defer deleteDeploymentF()

			// Assert alerts
			alerts := getAlertsForPolicy(t, alertService)
			assert.Len(t, alerts, 1)

			// NetworkPolicy creation
			deleteNetworkPolicyF := applyNetworkPolicy(t)
			defer deleteNetworkPolicyF()

			// TODO(ROX-9824): This can be removed once policies are evaluated on NetworkPolicy updates.
			scaleDeployment(t, netpolDeploymentName, "4")

			// Assert alerts
			alerts = getAlertsForPolicy(t, alertService)
			assert.Len(t, alerts, 0)
		})
	}

}
