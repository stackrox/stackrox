package reportgenerator

import (
	"fmt"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

func testNamespaces(clusters []*storage.Cluster, namespacesPerCluster int) []*storage.NamespaceMetadata {
	namespaces := make([]*storage.NamespaceMetadata, 0)
	for _, cluster := range clusters {
		for i := 0; i < namespacesPerCluster; i++ {
			namespaceName := fmt.Sprintf("ns%d", i+1)
			namespaces = append(namespaces, &storage.NamespaceMetadata{
				Id:          uuid.NewV4().String(),
				Name:        namespaceName,
				ClusterId:   cluster.Id,
				ClusterName: cluster.Name,
			})
		}
	}
	return namespaces
}

func allSeverities() []storage.VulnerabilitySeverity {
	return []storage.VulnerabilitySeverity{
		storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
	}
}

func testDeploymentsWithImages(namespaces []*storage.NamespaceMetadata, numDeploymentsPerNamespace int) ([]*storage.Deployment, []*storage.Image) {
	capacity := len(namespaces) * numDeploymentsPerNamespace
	deployments := make([]*storage.Deployment, 0, capacity)
	images := make([]*storage.Image, 0, capacity)

	for _, namespace := range namespaces {
		for i := 0; i < numDeploymentsPerNamespace; i++ {
			depName := fmt.Sprintf("%s_%s_dep%d", namespace.ClusterName, namespace.Name, i)
			image := testImage(depName)
			deployment := testDeployment(depName, namespace, image)
			deployments = append(deployments, deployment)
			images = append(images, image)
		}
	}
	return deployments, images
}

func testDeployment(deploymentName string, namespace *storage.NamespaceMetadata, image *storage.Image) *storage.Deployment {
	return &storage.Deployment{
		Name:        deploymentName,
		Id:          uuid.NewV4().String(),
		ClusterName: namespace.ClusterName,
		ClusterId:   namespace.ClusterId,
		Namespace:   namespace.Name,
		NamespaceId: namespace.Id,
		Containers: []*storage.Container{
			{
				Name:  fmt.Sprintf("%s_container", deploymentName),
				Image: imageTypes.ToContainerImage(image),
			},
		},
	}
}

func testWatchedImages(numImages int) []*storage.Image {
	images := make([]*storage.Image, 0, numImages)
	for i := 0; i < numImages; i++ {
		imgNamePrefix := fmt.Sprintf("w%d", i)
		image := testImage(imgNamePrefix)
		images = append(images, image)
	}
	return images
}

func testImage(prefix string) *storage.Image {
	t, err := ptypes.TimestampProto(time.Unix(0, 1000))
	utils.CrashOnError(err)
	return &storage.Image{
		Id:   fmt.Sprintf("%s_img", prefix),
		Name: &storage.ImageName{FullName: fmt.Sprintf("%s_img", prefix)},
		SetComponents: &storage.Image_Components{
			Components: 1,
		},
		SetCves: &storage.Image_Cves{
			Cves: 2,
		},
		Scan: &storage.ImageScan{
			ScanTime: t,
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    fmt.Sprintf("%s_img_comp", prefix),
					Version: "1.0",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve: fmt.Sprintf("CVE-fixable_critical-%s_img_comp", prefix),
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.1",
							},
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							Link:     "link",
						},
						{
							Cve:      fmt.Sprintf("CVE-nonFixable_low-%s_img_comp", prefix),
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
							Link:     "link",
						},
					},
				},
			},
		},
	}
}

func testCollection(collectionName, cluster, namespace, deployment string) *storage.ResourceCollection {
	collection := &storage.ResourceCollection{
		Name: collectionName,
		ResourceSelectors: []*storage.ResourceSelector{
			{
				Rules: []*storage.SelectorRule{},
			},
		},
	}
	if cluster != "" {
		collection.ResourceSelectors[0].Rules = append(collection.ResourceSelectors[0].Rules, &storage.SelectorRule{
			FieldName: pkgSearch.Cluster.String(),
			Operator:  storage.BooleanOperator_OR,
			Values: []*storage.RuleValue{
				{
					Value:     cluster,
					MatchType: storage.MatchType_EXACT,
				},
			},
		})
	}
	if namespace != "" {
		collection.ResourceSelectors[0].Rules = append(collection.ResourceSelectors[0].Rules, &storage.SelectorRule{
			FieldName: pkgSearch.Namespace.String(),
			Operator:  storage.BooleanOperator_OR,
			Values: []*storage.RuleValue{
				{
					Value:     namespace,
					MatchType: storage.MatchType_EXACT,
				},
			},
		})
	}
	var deploymentVal string
	var matchType storage.MatchType
	if deployment != "" {
		deploymentVal = deployment
		matchType = storage.MatchType_EXACT
	} else {
		deploymentVal = ".*"
		matchType = storage.MatchType_REGEX
	}
	collection.ResourceSelectors[0].Rules = append(collection.ResourceSelectors[0].Rules, &storage.SelectorRule{
		FieldName: pkgSearch.DeploymentName.String(),
		Operator:  storage.BooleanOperator_OR,
		Values: []*storage.RuleValue{
			{
				Value:     deploymentVal,
				MatchType: matchType,
			},
		},
	})

	return collection
}

func testReportSnapshot(collectionID string,
	fixability storage.VulnerabilityReportFilters_Fixability,
	severities []storage.VulnerabilitySeverity,
	imageTypes []storage.VulnerabilityReportFilters_ImageType,
	scopeRules []*storage.SimpleAccessScope_Rules) *storage.ReportSnapshot {
	snap := fixtures.GetReportSnapshot()
	snap.Filter = &storage.ReportSnapshot_VulnReportFilters{
		VulnReportFilters: &storage.VulnerabilityReportFilters{
			Fixability: fixability,
			Severities: severities,
			ImageTypes: imageTypes,
			CvesSince: &storage.VulnerabilityReportFilters_AllVuln{
				AllVuln: true,
			},
			AccessScopeRules: scopeRules,
		},
	}
	snap.Collection = &storage.CollectionSnapshot{
		Id:   collectionID,
		Name: collectionID,
	}
	return snap
}
