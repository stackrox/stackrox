package main

import (
	"bytes"
	"os"
	"reflect"

	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/central/graphql/generator"
	"github.com/stackrox/rox/central/graphql/generator/codegen"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

var (
	walkParameters = generator.TypeWalkParameters{
		IncludedTypes: []reflect.Type{
			reflect.TypeFor[*storage.Alert](),
			reflect.TypeFor[*storage.Cluster](),
			reflect.TypeFor[*storage.ClusterCVE](),
			reflect.TypeFor[*storage.ComplianceAggregation_Response](),
			reflect.TypeFor[*storage.ComplianceControlResult](),
			reflect.TypeFor[*storage.CVE](),
			reflect.TypeFor[*storage.Deployment](),
			reflect.TypeFor[*storage.FalsePositiveRequest](),
			reflect.TypeFor[*storage.Group](),
			reflect.TypeFor[*storage.Image](),
			reflect.TypeFor[*storage.ImageV2](),
			reflect.TypeFor[*storage.K8SRole](),
			reflect.TypeFor[*storage.K8SRoleBinding](),
			reflect.TypeFor[*storage.ListAlert](),
			reflect.TypeFor[*storage.ListDeployment](),
			reflect.TypeFor[*storage.ListImage](),
			reflect.TypeFor[*storage.ListImageV2](),
			reflect.TypeFor[*storage.ListSecret](),
			reflect.TypeFor[*storage.MitreAttackVector](),
			reflect.TypeFor[*storage.NetworkFlow](),
			reflect.TypeFor[*storage.Node](),
			reflect.TypeFor[*storage.NodeComponent](),
			reflect.TypeFor[*storage.NodeCVE](),
			reflect.TypeFor[*storage.Notifier](),
			reflect.TypeFor[*storage.PermissionSet](),
			reflect.TypeFor[*storage.Pod](),
			reflect.TypeFor[*storage.RequestComment](),
			reflect.TypeFor[*storage.Risk](),
			reflect.TypeFor[*storage.Role](),
			reflect.TypeFor[*storage.Secret](),
			reflect.TypeFor[*storage.ServiceAccount](),
			reflect.TypeFor[*storage.SimpleAccessScope](),
			reflect.TypeFor[*storage.SlimUser](),
			reflect.TypeFor[*storage.Subject](),
			reflect.TypeFor[*storage.TokenMetadata](),
			reflect.TypeFor[*storage.VulnerabilityRequest_Scope](),
			reflect.TypeFor[*storage.VulnerabilityRequest_CVEs](),
			reflect.TypeFor[*storage.ComplianceDomain_Cluster](),
			reflect.TypeFor[*storage.ComplianceDomain_Deployment](),
			reflect.TypeFor[*storage.ComplianceDomain_Node](),

			reflect.TypeFor[*v1.ComplianceStandard](),
			reflect.TypeFor[*v1.GenerateTokenResponse](),
			reflect.TypeFor[*v1.GetComplianceRunStatusesResponse](),
			reflect.TypeFor[*v1.GetPermissionsResponse](),
			reflect.TypeFor[*v1.Metadata](),
			reflect.TypeFor[*v1.Namespace](),
			reflect.TypeFor[*v1.ProcessNameGroup](),
			reflect.TypeFor[*v1.ScopeObject](),
			reflect.TypeFor[*v1.SearchResult](),
		},
		SkipResolvers: []reflect.Type{
			reflect.TypeFor[storage.BaseImageInfo](),
			reflect.TypeFor[storage.EmbeddedVulnerability](),
			reflect.TypeFor[storage.EmbeddedImageScanComponent](),
			reflect.TypeFor[storage.EmbeddedNodeScanComponent](),
			protocompat.TimestampType,
			reflect.TypeFor[storage.NodeVulnerability](),
			reflect.TypeFor[*storage.ImageCVEV2](),
			reflect.TypeFor[*storage.ImageComponentV2](),
			reflect.TypeFor[storage.EvaluationFilter](),
		},
		SkipFields: []generator.TypeAndField{
			{
				ParentType: reflect.TypeFor[storage.Image](),
				FieldName:  "Scan",
			},
			{
				ParentType: reflect.TypeFor[storage.Image](),
				FieldName:  "BaseImageInfo",
			},
			{
				ParentType: reflect.TypeFor[storage.ImageV2](),
				FieldName:  "Scan",
			},
			{
				ParentType: reflect.TypeFor[storage.ImageV2](),
				FieldName:  "ScanStats",
			},
			{
				ParentType: reflect.TypeFor[storage.ImageV2](),
				FieldName:  "BaseImageInfo",
			},
			{
				ParentType: reflect.TypeFor[storage.ImageScan](),
				FieldName:  "Components",
			},
			{
				ParentType: reflect.TypeFor[storage.NodeScan](),
				FieldName:  "Components",
			},
			{
				ParentType: reflect.TypeFor[storage.Node](),
				FieldName:  "Scan",
			},
			{
				ParentType: reflect.TypeFor[storage.CVE](),
				FieldName:  "Cvss",
			},
			{
				ParentType: reflect.TypeFor[storage.CVE](),
				FieldName:  "CvssV2",
			},
			{
				ParentType: reflect.TypeFor[storage.CVE](),
				FieldName:  "CvssV3",
			},
			{
				ParentType: reflect.TypeFor[storage.ImageComponentV2](),
				FieldName:  "Location",
			},
			{
				ParentType: reflect.TypeFor[storage.ImageSignatureVerificationResult](),
				FieldName:  "VerifierName",
			},
			{
				ParentType: reflect.TypeFor[storage.Policy](),
				FieldName:  "EvaluationFilter",
			},
			{
				ParentType: reflect.TypeFor[storage.ListPolicy](),
				FieldName:  "EvaluationFilter",
			},
		},
		InputTypes: []reflect.Type{
			reflect.TypeFor[*inputtypes.FalsePositiveVulnRequest](),
			reflect.TypeFor[*inputtypes.AggregateBy](),
			reflect.TypeFor[*inputtypes.SortOption](),
			reflect.TypeFor[*[]*inputtypes.SortOption](),
			reflect.TypeFor[*inputtypes.Pagination](),
			reflect.TypeFor[*inputtypes.VulnReqGlobalScope](),
			reflect.TypeFor[*inputtypes.VulnReqImageScope](),
			reflect.TypeFor[*inputtypes.VulnReqScope](),
			reflect.TypeFor[*analystnotes.ProcessNoteKey](),
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
