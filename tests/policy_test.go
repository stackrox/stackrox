package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
)

const (
	knownPolicyID = "d3e480c1-c6de-4cd2-9006-9a3eb3ad36b6"
	notAnID       = "Joseph Rules"
)

func TestPolicies(t *testing.T) {
	assumeFeatureFlagHasValue(t, features.PolicyImportExport, true)
	verifyExportNonExistentFails(t)
	verifyExportExistentSucceeds(t)
	verifyMixedExportFails(t)
}

func compareErrorsToExpected(t *testing.T, expectedErrors []*v1.ExportPolicyError, apiError error) {
	apiStatus, ok := status.FromError(apiError)
	require.True(t, ok)
	details := apiStatus.Details()
	require.Len(t, details, 1)
	exportErrors, ok := details[0].(*v1.ExportPoliciesErrorList)
	require.True(t, ok)
	// actual errors == expected errors ignoring order
	require.Len(t, exportErrors.GetErrors(), len(expectedErrors))
	for _, expected := range expectedErrors {
		require.Contains(t, exportErrors.GetErrors(), expected)
	}
}

func makeError(errorID, errorString string) *v1.ExportPolicyError {
	return &v1.ExportPolicyError{
		PolicyId: errorID,
		Error: &v1.PolicyError{
			Error: errorString,
		},
	}
}

func verifyExportNonExistentFails(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

	mockErrors := []*v1.ExportPolicyError{
		makeError(notAnID, "not found"),
	}
	service := v1.NewPolicyServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.ExportPolicies(ctx, &v1.ExportPoliciesRequest{
		PolicyIds: []string{notAnID},
	})
	cancel()
	require.Nil(t, resp)
	require.Error(t, err)
	compareErrorsToExpected(t, mockErrors, err)
}

func verifyExportExistentSucceeds(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

	service := v1.NewPolicyServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.ExportPolicies(ctx, &v1.ExportPoliciesRequest{
		PolicyIds: []string{knownPolicyID},
	})
	cancel()
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.GetPolicies(), 1)
	require.Equal(t, knownPolicyID, resp.GetPolicies()[0].GetId())
}

func verifyMixedExportFails(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

	mockErrors := []*v1.ExportPolicyError{
		makeError(notAnID, "not found"),
	}
	service := v1.NewPolicyServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.ExportPolicies(ctx, &v1.ExportPoliciesRequest{
		PolicyIds: []string{knownPolicyID, notAnID},
	})
	cancel()
	require.Nil(t, resp)
	require.Error(t, err)
	compareErrorsToExpected(t, mockErrors, err)
}
