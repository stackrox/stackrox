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
			m: &storage.SimpleAccessScope{
				Id:   testAccessScopeID,
				Name: testAccessScopeName,
			},
			expectedID:   testAccessScopeID,
			expectedName: testAccessScopeName,
		},
		"AuthProvider": {
			m: &storage.AuthProvider{
				Id:   testAuthProviderID,
				Name: testAuthProviderName,
			},
			expectedID:   testAuthProviderID,
			expectedName: testAuthProviderName,
		},
		"AuthMachineToMachineConfig": {
			m: &storage.AuthMachineToMachineConfig{
				Id:     testAuthM2MConfigID,
				Issuer: testAuthM2MConfigIssuer,
			},
			expectedID:   testAuthM2MConfigID,
			expectedName: testAuthM2MConfigIssuer,
		},
		"Deployment": {
			m: &storage.Deployment{
				Id:   testDeploymentID,
				Name: testDeploymentName,
			},
			expectedID:   testDeploymentID,
			expectedName: testDeploymentName,
		},
		"Group": {
			m: &storage.Group{
				Props: &storage.GroupProperties{
					Id:             testGroupID,
					Traits:         nil,
					AuthProviderId: testAuthProviderID,
					Key:            "key",
					Value:          "value",
				},
				RoleName: testRoleName,
			},
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
			m: &storage.Notifier{
				Id:   testNotifierID,
				Name: testNotifierName,
			},
			expectedID:   testNotifierID,
			expectedName: testNotifierName,
		},
		"PermissionSet": {
			m: &storage.PermissionSet{
				Id:   testPermissionSetID,
				Name: testPermissionSetName,
			},
			expectedID:   testPermissionSetID,
			expectedName: testPermissionSetName,
		},
		"Role": {
			m: &storage.Role{
				Name: testRoleName,
			},
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
