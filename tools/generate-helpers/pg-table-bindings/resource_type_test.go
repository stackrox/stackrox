package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stretchr/testify/assert"
)

func TestGetResourceType(t *testing.T) {
	for _, tc := range []struct {
		resourceType      ResourceType
		typ               proto.Message
		permissionChecker bool
		joinTable         bool
	}{
		{typ: &storage.NamespaceMetadata{}, resourceType: directlyScoped},
		{typ: &storage.NamespaceMetadata{}, resourceType: joinTable, joinTable: true},
		{typ: &storage.NamespaceMetadata{}, resourceType: joinTable, joinTable: true, permissionChecker: true},
		{typ: &storage.NamespaceMetadata{}, resourceType: permissionChecker, joinTable: false, permissionChecker: true},
		{typ: &storage.Cluster{}, resourceType: directlyScoped},
		{typ: &storage.Deployment{}, resourceType: directlyScoped},
		{typ: &storage.Image{}, resourceType: indirectlyScoped},
		{typ: &storage.CVE{}, resourceType: indirectlyScoped},
		{typ: &storage.Policy{}, resourceType: globallyScoped},
		{typ: &storage.Email{}, resourceType: joinTable, joinTable: true},
		{typ: &storage.Email{}, resourceType: permissionChecker, permissionChecker: true},
		// Evaluate types of the various stores
		{typ: &storage.ActiveComponent{}, resourceType: indirectlyScoped},
		{typ: &storage.Alert{}, resourceType: directlyScoped},
		{typ: &storage.AuthProvider{}, resourceType: globallyScoped},
		{typ: &storage.Blob{}, resourceType: globallyScoped},
		{typ: &storage.Cluster{}, resourceType: directlyScoped},
		{typ: &storage.ClusterHealthStatus{}, resourceType: indirectlyScoped},
		{typ: &storage.ClusterCVE{}, resourceType: indirectlyScoped},
		{typ: &storage.ClusterCVEEdge{}, resourceType: indirectlyScoped},
		{typ: &storage.ComplianceConfig{}, resourceType: indirectlyScoped},
		{typ: &storage.ComplianceControlResult{}, resourceType: indirectlyScoped},
		{typ: &storage.ComplianceDomain{}, resourceType: indirectlyScoped},
		{typ: &storage.ComplianceIntegration{}, resourceType: directlyScoped},
		{typ: &storage.ComplianceOperatorCheckResult{}, resourceType: globallyScoped},
		{typ: &storage.ComplianceOperatorCheckResultV2{}, resourceType: directlyScoped},
		{typ: &storage.ComplianceOperatorClusterScanConfigStatus{}, resourceType: directlyScoped},
		{typ: &storage.ComplianceOperatorProfile{}, resourceType: globallyScoped},
		{typ: &storage.ComplianceOperatorProfileV2{}, resourceType: directlyScoped},
		{typ: &storage.ComplianceOperatorRule{}, resourceType: globallyScoped},
		{typ: &storage.ComplianceOperatorRuleV2{}, resourceType: directlyScoped},
		{typ: &storage.ComplianceOperatorScan{}, resourceType: globallyScoped},
		{typ: &storage.ComplianceOperatorScanV2{}, resourceType: directlyScoped},
		{typ: &storage.ComplianceOperatorScanConfigurationV2{}, resourceType: indirectlyScoped},
		{typ: &storage.ComplianceOperatorScanSettingBinding{}, resourceType: globallyScoped},
		{typ: &storage.ComplianceOperatorSuite{}, resourceType: directlyScoped},
		{typ: &storage.ComplianceRunMetadata{}, resourceType: directlyScoped},
		{typ: &storage.ComplianceRunResults{}, resourceType: directlyScoped},
		{typ: &storage.ComplianceStrings{}, resourceType: indirectlyScoped},
		{typ: &storage.ComponentCVEEdge{}, resourceType: indirectlyScoped},
		{typ: &storage.Config{}, resourceType: globallyScoped},
		{typ: &storage.CVE{}, resourceType: indirectlyScoped},
		{typ: &storage.DeclarativeConfigHealth{}, resourceType: globallyScoped},
		{typ: &storage.DelegatedRegistryConfig{}, resourceType: globallyScoped},
		{typ: &storage.Deployment{}, resourceType: directlyScoped},
		{typ: &storage.ExternalBackup{}, resourceType: globallyScoped},
		{typ: &storage.Group{}, resourceType: globallyScoped},
		{typ: &storage.Hash{}, resourceType: globallyScoped},
		{typ: &storage.Image{}, resourceType: indirectlyScoped},
		{typ: &storage.ImageComponent{}, resourceType: indirectlyScoped},
		{typ: &storage.ImageComponentEdge{}, resourceType: indirectlyScoped},
		{typ: &storage.ImageCVE{}, resourceType: indirectlyScoped},
		{typ: &storage.ImageCVEEdge{}, resourceType: indirectlyScoped},
		{typ: &storage.ImageIntegration{}, resourceType: globallyScoped},
		{typ: &storage.InitBundleMeta{}, resourceType: permissionChecker, permissionChecker: true}, // globallyScoped
		{typ: &storage.IntegrationHealth{}, resourceType: globallyScoped},
		{typ: &storage.K8SRole{}, resourceType: directlyScoped},
		{typ: &storage.K8SRoleBinding{}, resourceType: directlyScoped},
		{typ: &storage.LogImbue{}, resourceType: globallyScoped},
		{typ: &storage.NamespaceMetadata{}, resourceType: directlyScoped},
		{typ: &storage.NetworkBaseline{}, resourceType: directlyScoped},
		{typ: &storage.NetworkEntity{}, resourceType: indirectlyScoped},
		{typ: &storage.NetworkFlow{}, resourceType: indirectlyScoped},
		{typ: &storage.NetworkGraphConfig{}, resourceType: globallyScoped},
		{typ: &storage.NetworkPolicy{}, resourceType: directlyScoped},
		{typ: &storage.NetworkPolicyApplicationUndoDeploymentRecord{}, resourceType: indirectlyScoped},
		{typ: &storage.NetworkPolicyApplicationUndoRecord{}, resourceType: indirectlyScoped},
		{typ: &storage.Node{}, resourceType: directlyScoped},
		{typ: &storage.NodeComponent{}, resourceType: indirectlyScoped},
		{typ: &storage.NodeComponentCVEEdge{}, resourceType: indirectlyScoped},
		{typ: &storage.NodeComponentEdge{}, resourceType: indirectlyScoped},
		{typ: &storage.NodeCVE{}, resourceType: indirectlyScoped},
		{typ: &storage.NotificationSchedule{}, resourceType: globallyScoped},
		{typ: &storage.Notifier{}, resourceType: globallyScoped},
		{typ: &storage.PermissionSet{}, resourceType: globallyScoped},
		{typ: &storage.Pod{}, resourceType: directlyScoped},
		{typ: &storage.Policy{}, resourceType: globallyScoped},
		{typ: &storage.PolicyCategory{}, resourceType: globallyScoped},
		{typ: &storage.PolicyCategoryEdge{}, resourceType: globallyScoped},
		{typ: &storage.ProcessBaseline{}, resourceType: directlyScoped},
		{typ: &storage.ProcessBaselineResults{}, resourceType: directlyScoped},
		{typ: &storage.ProcessIndicator{}, resourceType: directlyScoped},
		{typ: &storage.ProcessListeningOnPortStorage{}, resourceType: directlyScoped},
		{typ: &storage.ReportConfiguration{}, resourceType: globallyScoped},
		{typ: &storage.ReportSnapshot{}, resourceType: globallyScoped},
		{typ: &storage.ResourceCollection{}, resourceType: globallyScoped},
		{typ: &storage.Risk{}, resourceType: directlyScoped},
		{typ: &storage.Role{}, resourceType: globallyScoped},
		{typ: &storage.Secret{}, resourceType: directlyScoped},
		{typ: &storage.SensorUpgradeConfig{}, resourceType: globallyScoped},
		{typ: &storage.ServiceAccount{}, resourceType: directlyScoped},
		{typ: &storage.ServiceIdentity{}, resourceType: globallyScoped},
		{typ: &storage.SignatureIntegration{}, resourceType: globallyScoped},
		{typ: &storage.SimpleAccessScope{}, resourceType: globallyScoped},
		{typ: &storage.TelemetryConfiguration{}, resourceType: globallyScoped},
		{typ: &storage.TokenMetadata{}, resourceType: globallyScoped},
		{typ: &storage.User{}, resourceType: globallyScoped},
		{typ: &storage.WatchedImage{}, resourceType: globallyScoped},
	} {
		tc := tc
		t.Run(fmt.Sprintf("%T (join: %t, perm: %t) -> %d", tc.typ, tc.joinTable, tc.permissionChecker, tc.resourceType), func(t *testing.T) {
			actual := getResourceType(
				fmt.Sprintf("%T", tc.typ),
				walker.Walk(reflect.TypeOf(tc.typ), ""),
				tc.permissionChecker,
				tc.joinTable,
			)
			assert.Equal(t, tc.resourceType, actual)
		})
	}

	t.Run("panics on unknown resource", func(t *testing.T) {
		email := &storage.Email{}
		assert.Panics(t, func() {
			getResourceType(fmt.Sprintf("%T", email), walker.Walk(reflect.TypeOf(email), ""), false, false)
		})
	})
}
