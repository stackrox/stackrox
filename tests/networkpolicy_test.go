package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
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
									Value: "false",
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

func getAlertsForPolicy(t testutils.T, client v1.AlertServiceClient, policyName string) []*storage.ListAlert {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var alerts []*storage.ListAlert
	resp, err := client.ListAlerts(ctx, &v1.ListAlertsRequest{
		Query: fmt.Sprintf("Policy:%s+Namespace:default", policyName),
	})

	if err != nil {
		t.Errorf("Failed to get alerts: %s", err)
		t.FailNow()
	}

	alerts = resp.GetAlerts()
	return alerts
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
	conn := centralgrpc.GRPCConnectionToCentral(t)
	if buildinfo.ReleaseBuild || !CheckIfCentralHasFeatureFlag(t, conn) {
		t.Skip("Feature flag disabled")
	}

	testCases := map[string]struct {
		policyName        string
		policyField       string
		networkPolicyFile string
	}{
		"Should have ingress": {
			policyName:        "[Automated] Should have ingress policy",
			policyField:       "Has Ingress Network Policy",
			networkPolicyFile: "testdata/allow-ingress-netpol.yaml",
		},
		"Should have egress": {
			policyName:        "[Automated] Should have egress policy",
			policyField:       "Has Egress Network Policy",
			networkPolicyFile: "testdata/allow-egress-netpol.yaml",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {

			// This is to avoid flakyness in the tests.
			// I've observed that the `apply*` functions could fail during the waiting part and the
			// defer function is never registered. In case the code panics and litters the cluster
			// with stale test data, it's easier to simply delete them before running the test.
			teardownDeployment(t, netpolDeploymentName)
			teardownFile(t, testCase.networkPolicyFile)

			policyService := v1.NewPolicyServiceClient(conn)
			alertService := v1.NewAlertServiceClient(conn)

			// Policy creation
			deletePolicyF, err := createPolicyIfMissing(t, policyService, getFakePolicyRequest(testCase.policyName, testCase.policyField))
			defer deletePolicyF()
			assert.NoError(t, err)

			// Deployment creation
			setupDeploymentFromFile(t, netpolDeploymentName, "testdata/nginx.yaml")
			defer teardownDeploymentFromFile(t, netpolDeploymentName, "testdata/nginx.yaml")

			testutils.Retry(t, 3, 6*time.Second, func(retryT testutils.T) {
				// Assert alerts
				alerts := getAlertsForPolicy(retryT, alertService, testCase.policyName)
				assert.Len(retryT, alerts, 1)
			})

			// NetworkPolicy creation
			applyFile(t, testCase.networkPolicyFile)
			defer teardownFile(t, testCase.networkPolicyFile)

			// NetworkPolicy events do not trigger the immediate evaluation of deployments. The deployments are marked
			// for reprocessing and will be resynced every minute. To avoid having the tests wait for a minute to check
			// if the violation is present, we update the deployment to force a re-evaluation.
			scaleDeployment(t, netpolDeploymentName, "1")

			testutils.Retry(t, 3, 6*time.Second, func(retryT testutils.T) {
				// Assert alerts
				alerts := getAlertsForPolicy(retryT, alertService, testCase.policyName)
				assert.Len(retryT, alerts, 0)
			})
		})
	}

}
