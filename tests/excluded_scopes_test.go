//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/require"
)

func TestExcludedScopes(t *testing.T) {
	if os.Getenv("ORCHESTRATOR_FLAVOR") == "openshift" {
		t.Skip("temporarily skipped on OCP. TODO(ROX-25171)")
	}
	deploymentName := fmt.Sprintf("test-excluded-scopes-%d", rand.Intn(10000))

	setupDeploymentInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deploymentName, "default")
	defer teardownDeploymentWithoutCheck(t, deploymentName, "default")
	waitForDeploymentInCentral(t, deploymentName)

	verifyNoAlertForExcludedScopes(t, deploymentName)
	verifyAlertForExcludedScopesRemoval(t, deploymentName)
}

func verifyNoAlertForExcludedScopes(t *testing.T, deploymentName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewPolicyServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.ListPolicies(ctx, &v1.RawQuery{
		Query: search.NewQueryBuilder().AddStrings(search.PolicyName, expectedLatestTagPolicy).Query(),
	})
	cancel()
	require.NoError(t, err)
	require.Len(t, resp.GetPolicies(), 1)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	latestPolicy, err := service.GetPolicy(ctx, &v1.ResourceByID{
		Id: resp.GetPolicies()[0].GetId(),
	})
	cancel()
	require.NoError(t, err)

	latestPolicy.Exclusions = []*storage.Exclusion{
		{
			Deployment: &storage.Exclusion_Deployment{
				Name: deploymentName,
			},
		},
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = service.PutPolicy(ctx, latestPolicy)
	cancel()
	require.NoError(t, err)

	qb := search.NewQueryBuilder().AddStrings(search.DeploymentName, deploymentName).AddStrings(search.PolicyName, latestPolicy.GetName()).AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())
	alertService := v1.NewAlertServiceClient(conn)
	waitForAlert(t, alertService, &v1.ListAlertsRequest{
		Query: qb.Query(),
	}, 0)
}

func verifyAlertForExcludedScopesRemoval(t *testing.T, deploymentName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewPolicyServiceClient(conn)

	qb := search.NewQueryBuilder().AddStrings(search.PolicyName, expectedLatestTagPolicy)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.ListPolicies(ctx, &v1.RawQuery{
		Query: qb.Query(),
	})
	cancel()
	require.NoError(t, err)
	require.Len(t, resp.GetPolicies(), 1)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	latestPolicy, err := service.GetPolicy(ctx, &v1.ResourceByID{
		Id: resp.GetPolicies()[0].GetId(),
	})
	cancel()
	require.NoError(t, err)

	latestPolicy.Exclusions = nil
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = service.PutPolicy(ctx, latestPolicy)
	cancel()
	require.NoError(t, err)

	alertService := v1.NewAlertServiceClient(conn)

	qb = search.NewQueryBuilder().AddStrings(search.DeploymentName, deploymentName).AddStrings(search.PolicyName, latestPolicy.GetName()).AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())
	// Wait for alert to be removed now that it is excluded
	waitForAlert(t, alertService, &v1.ListAlertsRequest{
		Query: qb.Query(),
	}, 1)
}
