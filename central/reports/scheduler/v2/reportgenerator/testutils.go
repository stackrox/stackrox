package reportgenerator

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	types2 "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protocompat"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
)

func testNamespaces(clusters []*storage.Cluster, namespacesPerCluster int) []*storage.NamespaceMetadata {
	namespaces := make([]*storage.NamespaceMetadata, 0)
	for _, cluster := range clusters {
		for i := 0; i < namespacesPerCluster; i++ {
			namespaceName := fmt.Sprintf("ns%d", i+1)
			nm := &storage.NamespaceMetadata{}
			nm.SetId(uuid.NewV4().String())
			nm.SetName(namespaceName)
			nm.SetClusterId(cluster.GetId())
			nm.SetClusterName(cluster.GetName())
			namespaces = append(namespaces, nm)
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

	for j, namespace := range namespaces {
		for i := 0; i < numDeploymentsPerNamespace; i++ {
			depName := fmt.Sprintf("%s_%s_dep%d", namespace.GetClusterName(), namespace.GetName(), i)
			image := testImage(depName)
			var deployment *storage.Deployment
			if j == 0 && i == 0 {
				// Add a copy of image with same SHA, components and CVEs, but different name
				image2 := image.CloneVT()
				image2.GetName().SetFullName(image.GetName().GetFullName() + "_copy")
				deployment = testDeployment(depName, namespace, image, image2)
			} else {
				deployment = testDeployment(depName, namespace, image)
			}
			deployments = append(deployments, deployment)
			images = append(images, image)
		}
	}
	return deployments, images
}

func testDeployment(deploymentName string, namespace *storage.NamespaceMetadata, images ...*storage.Image) *storage.Deployment {
	dep := &storage.Deployment{}
	dep.SetName(deploymentName)
	dep.SetId(uuid.NewV4().String())
	dep.SetClusterName(namespace.GetClusterName())
	dep.SetClusterId(namespace.GetClusterId())
	dep.SetNamespace(namespace.GetName())
	dep.SetNamespaceId(namespace.GetId())

	containers := make([]*storage.Container, 0, len(images))
	for i, image := range images {
		container := &storage.Container{}
		container.SetName(fmt.Sprintf("%s_container_%d", deploymentName, i))
		container.SetImage(types2.ToContainerImage(image))
		containers = append(containers, container)
	}
	dep.SetContainers(containers)
	return dep
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
	t, err := protocompat.ConvertTimeToTimestampOrError(time.Unix(0, 1000))
	utils.CrashOnError(err)
	cVSSV3 := &storage.CVSSV3{}
	cVSSV3.SetScore(10)
	nvdCvss := &storage.CVSSScore{}
	nvdCvss.SetSource(storage.Source_SOURCE_NVD)
	nvdCvss.SetCvssv3(proto.ValueOrDefault(cVSSV3))
	return storage.Image_builder{
		Id: fmt.Sprintf("%s_img", prefix),
		Name: storage.ImageName_builder{
			FullName: fmt.Sprintf("%s_img", prefix),
			Registry: "docker.io",
			Remote:   fmt.Sprintf("library/%s_img", prefix),
			Tag:      "latest",
		}.Build(),
		Components: proto.Int32(1),
		Cves:       proto.Int32(2),
		Scan: storage.ImageScan_builder{
			ScanTime: t,
			Components: []*storage.EmbeddedImageScanComponent{
				storage.EmbeddedImageScanComponent_builder{
					Name:     fmt.Sprintf("%s_img_comp", prefix),
					Version:  "1.0",
					Source:   storage.SourceType_OS,
					Location: "/usr/lib",
					Vulns: []*storage.EmbeddedVulnerability{
						storage.EmbeddedVulnerability_builder{
							Cve:         fmt.Sprintf("CVE-fixable_critical-%s_img_comp", prefix),
							FixedBy:     proto.String("1.1"),
							CvssMetrics: []*storage.CVSSScore{nvdCvss},
							Advisory: storage.Advisory_builder{
								Name: "RHSA-2025-CVE-fixable",
								Link: "test-rhsa-link",
							}.Build(),
							Severity:              storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							Link:                  "link",
							Cvss:                  9.0,
							State:                 storage.VulnerabilityState_OBSERVED,
							FirstSystemOccurrence: t,
							FirstImageOccurrence:  t,
							NvdCvss:               8.5,
							Epss: storage.EPSS_builder{
								EpssProbability: 0.7,
								EpssPercentile:  0.8,
							}.Build(),
							CvssV2: storage.CVSSV2_builder{
								Vector:              "AV:N/AC:L/Au:N/C:P/I:P/A:P",
								Score:               7.5,
								ExploitabilityScore: 10.0,
								ImpactScore:         6.4,
							}.Build(),
							CvssV3: storage.CVSSV3_builder{
								Vector:              "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
								Score:               9.8,
								ExploitabilityScore: 3.9,
								ImpactScore:         5.9,
							}.Build(),
						}.Build(),
						storage.EmbeddedVulnerability_builder{
							Cve:      fmt.Sprintf("CVE-nonFixable_low-%s_img_comp", prefix),
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
							Link:     "link",
							Advisory: storage.Advisory_builder{
								Name: "RHSA-2025-CVE-fixable",
								Link: "test-rhsa-link",
							}.Build(),
							Cvss:                  2.0,
							State:                 storage.VulnerabilityState_OBSERVED,
							FirstSystemOccurrence: t,
							FirstImageOccurrence:  t,
							NvdCvss:               1.8,
							Epss: storage.EPSS_builder{
								EpssProbability: 0.1,
								EpssPercentile:  0.2,
							}.Build(),
							CvssV2: storage.CVSSV2_builder{
								Vector:              "AV:L/AC:H/Au:N/C:P/I:N/A:N",
								Score:               1.9,
								ExploitabilityScore: 1.9,
								ImpactScore:         2.9,
							}.Build(),
							CvssV3: storage.CVSSV3_builder{
								Vector:              "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:L/I:N/A:N",
								Score:               2.3,
								ExploitabilityScore: 1.0,
								ImpactScore:         1.4,
							}.Build(),
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
	}.Build()
}

func testCollection(collectionName, cluster, namespace, deployment string) *storage.ResourceCollection {
	collection := storage.ResourceCollection_builder{
		Name: collectionName,
		ResourceSelectors: []*storage.ResourceSelector{
			storage.ResourceSelector_builder{
				Rules: []*storage.SelectorRule{},
			}.Build(),
		},
	}.Build()
	if cluster != "" {
		ruleValue := &storage.RuleValue{}
		ruleValue.SetValue(cluster)
		ruleValue.SetMatchType(storage.MatchType_EXACT)
		sr := &storage.SelectorRule{}
		sr.SetFieldName(pkgSearch.Cluster.String())
		sr.SetOperator(storage.BooleanOperator_OR)
		sr.SetValues([]*storage.RuleValue{
			ruleValue,
		})
		collection.GetResourceSelectors()[0].SetRules(append(collection.GetResourceSelectors()[0].GetRules(), sr))
	}
	if namespace != "" {
		ruleValue := &storage.RuleValue{}
		ruleValue.SetValue(namespace)
		ruleValue.SetMatchType(storage.MatchType_EXACT)
		sr := &storage.SelectorRule{}
		sr.SetFieldName(pkgSearch.Namespace.String())
		sr.SetOperator(storage.BooleanOperator_OR)
		sr.SetValues([]*storage.RuleValue{
			ruleValue,
		})
		collection.GetResourceSelectors()[0].SetRules(append(collection.GetResourceSelectors()[0].GetRules(), sr))
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
	ruleValue := &storage.RuleValue{}
	ruleValue.SetValue(deploymentVal)
	ruleValue.SetMatchType(matchType)
	sr := &storage.SelectorRule{}
	sr.SetFieldName(pkgSearch.DeploymentName.String())
	sr.SetOperator(storage.BooleanOperator_OR)
	sr.SetValues([]*storage.RuleValue{
		ruleValue,
	})
	collection.GetResourceSelectors()[0].SetRules(append(collection.GetResourceSelectors()[0].GetRules(), sr))

	return collection
}

func testReportSnapshot(collectionID string,
	fixability storage.VulnerabilityReportFilters_Fixability,
	severities []storage.VulnerabilitySeverity,
	imageTypes []storage.VulnerabilityReportFilters_ImageType,
	scopeRules []*storage.SimpleAccessScope_Rules) *storage.ReportSnapshot {
	snap := fixtures.GetReportSnapshot()
	vrf := &storage.VulnerabilityReportFilters{}
	vrf.SetFixability(fixability)
	vrf.SetSeverities(severities)
	vrf.SetImageTypes(imageTypes)
	vrf.SetAllVuln(true)
	vrf.SetAccessScopeRules(scopeRules)
	snap.SetVulnReportFilters(proto.ValueOrDefault(vrf))
	cs := &storage.CollectionSnapshot{}
	cs.SetId(collectionID)
	cs.SetName(collectionID)
	snap.SetCollection(cs)
	return snap
}

func testViewBasedReportSnapshot(query string, scopeRules []*storage.SimpleAccessScope_Rules) *storage.ReportSnapshot {
	snap := fixtures.GetReportSnapshot()
	vbvrf := &storage.ViewBasedVulnerabilityReportFilters{}
	vbvrf.SetQuery(query)
	vbvrf.SetAccessScopeRules(scopeRules)
	snap.SetViewBasedVulnReportFilters(proto.ValueOrDefault(vbvrf))
	return snap
}
