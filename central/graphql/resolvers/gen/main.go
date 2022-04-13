package main

import (
	"bytes"
	"os"
	"reflect"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/central/graphql/generator"
	"github.com/stackrox/rox/central/graphql/generator/codegen"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

var (
	walkParameters = generator.TypeWalkParameters{
		IncludedTypes: []reflect.Type{
			reflect.TypeOf((*storage.ActiveComponent_ActiveContext)(nil)),
			reflect.TypeOf((*storage.Alert)(nil)),
			reflect.TypeOf((*storage.Cluster)(nil)),
			reflect.TypeOf((*storage.ClusterCVE)(nil)),
			reflect.TypeOf((*storage.ComplianceAggregation_Response)(nil)),
			reflect.TypeOf((*storage.ComplianceControlResult)(nil)),
			reflect.TypeOf((*storage.CVE)(nil)),
			reflect.TypeOf((*storage.Deployment)(nil)),
			reflect.TypeOf((*storage.FalsePositiveRequest)(nil)),
			reflect.TypeOf((*storage.Group)(nil)),
			reflect.TypeOf((*storage.Image)(nil)),
			reflect.TypeOf((*storage.ImageComponent)(nil)),
			reflect.TypeOf((*storage.ImageCVE)(nil)),
			reflect.TypeOf((*storage.K8SRole)(nil)),
			reflect.TypeOf((*storage.K8SRoleBinding)(nil)),
			reflect.TypeOf((*storage.ListAlert)(nil)),
			reflect.TypeOf((*storage.ListDeployment)(nil)),
			reflect.TypeOf((*storage.ListImage)(nil)),
			reflect.TypeOf((*storage.ListSecret)(nil)),
			reflect.TypeOf((*storage.MitreAttackVector)(nil)),
			reflect.TypeOf((*storage.NetworkFlow)(nil)),
			reflect.TypeOf((*storage.Node)(nil)),
			reflect.TypeOf((*storage.NodeComponent)(nil)),
			reflect.TypeOf((*storage.NodeCVE)(nil)),
			reflect.TypeOf((*storage.Notifier)(nil)),
			reflect.TypeOf((*storage.PermissionSet)(nil)),
			reflect.TypeOf((*storage.Pod)(nil)),
			reflect.TypeOf((*storage.RequestComment)(nil)),
			reflect.TypeOf((*storage.Risk)(nil)),
			reflect.TypeOf((*storage.Role)(nil)),
			reflect.TypeOf((*storage.Secret)(nil)),
			reflect.TypeOf((*storage.ServiceAccount)(nil)),
			reflect.TypeOf((*storage.SimpleAccessScope)(nil)),
			reflect.TypeOf((*storage.SlimUser)(nil)),
			reflect.TypeOf((*storage.Subject)(nil)),
			reflect.TypeOf((*storage.TokenMetadata)(nil)),
			reflect.TypeOf((*storage.VulnerabilityRequest_Scope)(nil)),
			reflect.TypeOf((*storage.VulnerabilityRequest_CVEs)(nil)),
			reflect.TypeOf((*storage.ComplianceDomain_Cluster)(nil)),
			reflect.TypeOf((*storage.ComplianceDomain_Deployment)(nil)),
			reflect.TypeOf((*storage.ComplianceDomain_Node)(nil)),

			reflect.TypeOf((*v1.ComplianceRunScheduleInfo)(nil)),
			reflect.TypeOf((*v1.ComplianceStandard)(nil)),
			reflect.TypeOf((*v1.GenerateTokenResponse)(nil)),
			reflect.TypeOf((*v1.GetComplianceRunStatusesResponse)(nil)),
			reflect.TypeOf((*v1.GetPermissionsResponse)(nil)),
			reflect.TypeOf((*v1.Metadata)(nil)),
			reflect.TypeOf((*v1.Namespace)(nil)),
			reflect.TypeOf((*v1.ProcessNameGroup)(nil)),
			reflect.TypeOf((*v1.SearchResult)(nil)),
		},
		SkipResolvers: []reflect.Type{
			reflect.TypeOf(storage.EmbeddedVulnerability{}),
			reflect.TypeOf(storage.EmbeddedImageScanComponent{}),
			reflect.TypeOf(storage.EmbeddedNodeScanComponent{}),
			reflect.TypeOf(types.Timestamp{}),
			reflect.TypeOf(storage.NodeVulnerability{}),
		},
		SkipFields: []generator.TypeAndField{
			{
				ParentType: reflect.TypeOf(storage.Image{}),
				FieldName:  "Scan",
			},
			{
				ParentType: reflect.TypeOf(storage.ImageScan{}),
				FieldName:  "Components",
			},
			{
				ParentType: reflect.TypeOf(storage.NodeScan{}),
				FieldName:  "Components",
			},
			{
				ParentType: reflect.TypeOf(storage.Node{}),
				FieldName:  "Scan",
			},
			// TODO(ROX-6194): Remove this entirely after the deprecation cycle started with the 55.0 release.
			{
				ParentType: reflect.TypeOf(storage.Policy{}),
				FieldName:  "Whitelists",
			},

			{
				ParentType: reflect.TypeOf(storage.CVE{}),
				FieldName:  "Cvss",
			},
			{
				ParentType: reflect.TypeOf(storage.CVE{}),
				FieldName:  "CvssV2",
			},
			{
				ParentType: reflect.TypeOf(storage.CVE{}),
				FieldName:  "CvssV3",
			},
		},
		InputTypes: []reflect.Type{
			reflect.TypeOf((*inputtypes.FalsePositiveVulnRequest)(nil)),
			reflect.TypeOf((*inputtypes.SortOption)(nil)),
			reflect.TypeOf((*inputtypes.Pagination)(nil)),
			reflect.TypeOf((*inputtypes.VulnReqGlobalScope)(nil)),
			reflect.TypeOf((*inputtypes.VulnReqImageScope)(nil)),
			reflect.TypeOf((*inputtypes.VulnReqScope)(nil)),
			reflect.TypeOf((*analystnotes.ProcessNoteKey)(nil)),
		},
	}
)

func main() {
	w := &bytes.Buffer{}
	codegen.GenerateResolvers(walkParameters, w)
	err := os.WriteFile("generated.go", w.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}
