package types

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

const (
	testAccessScopeID   = "test access scope ID"
	testAuthProviderID  = "test auth provider ID"
	testAuthM2MConfigID = "test auth machine to machine configuration ID"
	testDeploymentID    = "test deployment ID"
	testGroupID         = "test group ID"
	testPermissionSetID = "test permission set ID"
	testNotifierID      = "test notifier ID"

	testAccessScopeName   = "test access scope name"
	testAuthProviderName  = "test auth provider name"
	testDeploymentName    = "test deployment name"
	testGroupName         = "test group name"
	testPermissionSetName = "test permission set name"
	testRoleName          = "test role name"
	testNotifierName      = "test notifier name"

	testAuthM2MConfigIssuer = "test auth machine to machine configuration issuer"
)

type testCase struct {
	m            protocompat.Message
	expectedID   string
	expectedName string
}

func getTestCases() map[string]testCase {
	return map[string]testCase{
		"AccessScope": {
			m: storage.SimpleAccessScope_builder{
				Id:   testAccessScopeID,
				Name: testAccessScopeName,
			}.Build(),
			expectedID:   testAccessScopeID,
			expectedName: testAccessScopeName,
		},
		"AuthProvider": {
			m: storage.AuthProvider_builder{
				Id:   testAuthProviderID,
				Name: testAuthProviderName,
			}.Build(),
			expectedID:   testAuthProviderID,
			expectedName: testAuthProviderName,
		},
		"AuthMachineToMachineConfig": {
			m: storage.AuthMachineToMachineConfig_builder{
				Id:     testAuthM2MConfigID,
				Issuer: testAuthM2MConfigIssuer,
			}.Build(),
			expectedID:   testAuthM2MConfigID,
			expectedName: testAuthM2MConfigIssuer,
		},
		"Deployment": {
			m: storage.Deployment_builder{
				Id:   testDeploymentID,
				Name: testDeploymentName,
			}.Build(),
			expectedID:   testDeploymentID,
			expectedName: testDeploymentName,
		},
		"Group": {
			m: storage.Group_builder{
				Props: storage.GroupProperties_builder{
					Id:             testGroupID,
					Traits:         nil,
					AuthProviderId: testAuthProviderID,
					Key:            "key",
					Value:          "value",
				}.Build(),
				RoleName: testRoleName,
			}.Build(),
			expectedID: testGroupID,
			expectedName: fmt.Sprintf(
				"group %s:%s:%s for auth provider ID %s",
				"key",
				"value",
				testRoleName,
				testAuthProviderID,
			),
		},
		"Notifier": {
			m: storage.Notifier_builder{
				Id:   testNotifierID,
				Name: testNotifierName,
			}.Build(),
			expectedID:   testNotifierID,
			expectedName: testNotifierName,
		},
		"PermissionSet": {
			m: storage.PermissionSet_builder{
				Id:   testPermissionSetID,
				Name: testPermissionSetName,
			}.Build(),
			expectedID:   testPermissionSetID,
			expectedName: testPermissionSetName,
		},
		"Role": {
			m: storage.Role_builder{
				Name: testRoleName,
			}.Build(),
			expectedID:   testRoleName,
			expectedName: testRoleName,
		},
	}
}

func TestExtractIDFromProtoMessage(t *testing.T) {
	for name, tc := range getTestCases() {
		t.Run(name, func(it *testing.T) {
			extracted := extractIDFromProtoMessage(tc.m)
			assert.Equal(it, tc.expectedID, extracted)
		})
	}
}

func TestExtractNameFromProtoMessage(t *testing.T) {
	for name, tc := range getTestCases() {
		t.Run(name, func(it *testing.T) {
			extracted := extractNameFromProtoMessage(tc.m)
			assert.Equal(it, tc.expectedName, extracted)
		})
	}
}
