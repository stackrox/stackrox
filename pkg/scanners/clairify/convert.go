package clairify

import (
	gogoProto "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/cvss/cvssv2"
	"github.com/stackrox/rox/pkg/cvss/cvssv3"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/scans"
	"github.com/stackrox/rox/pkg/stringutils"
	v1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

func convertNodeToVulnRequest(node *storage.Node) *v1.GetNodeVulnerabilitiesRequest {
	return &v1.GetNodeVulnerabilitiesRequest{
		OsImage:          node.GetOsImage(),
		KernelVersion:    node.GetKernelVersion(),
		KubeletVersion:   node.GetKubeletVersion(),
		KubeproxyVersion: node.GetKubeProxyVersion(),
		Runtime:          convertContainerRuntime(node.GetContainerRuntime()),
	}
}

func convertContainerRuntime(containerRuntime *storage.ContainerRuntimeInfo) *v1.GetNodeVulnerabilitiesRequest_ContainerRuntime {
	var name, version string
	switch containerRuntime.GetType() {
	case storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME:
		name = "docker"
		version = containerRuntime.GetVersion()
	case storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME:
		name = "cri-o"
		version = containerRuntime.GetVersion()
	default:
		runtime, v := stringutils.Split2(containerRuntime.GetVersion(), "://")
		if runtime != "containerd" && runtime != "runc" {
			log.Warnf("unsupported container runtime for node scanning: %s", runtime)
			return nil
		}
		name = runtime
		version = v
	}
	return &v1.GetNodeVulnerabilitiesRequest_ContainerRuntime{
		Name:    name,
		Version: version,
	}
}

func convertVulnResponseToNodeScan(req *v1.GetNodeVulnerabilitiesRequest, resp *v1.GetNodeVulnerabilitiesResponse) *storage.NodeScan {
	scan := &storage.NodeScan{
		ScanTime:        gogoProto.TimestampNow(),
		OperatingSystem: resp.GetOperatingSystem(),
		Components: []*storage.EmbeddedNodeScanComponent{
			{
				Name:    stringutils.OrDefault(resp.GetKernelComponent().GetName(), "kernel"),
				Version: resp.GetKernelComponent().GetVersion(),
				Vulns:   convertNodeVulns(resp.GetKernelVulnerabilities()),
			},
			{
				Name:    "kubelet",
				Version: req.GetKubeletVersion(),
				Vulns:   convertNodeVulns(resp.GetKubeletVulnerabilities()),
			},
			{
				Name:    "kube-proxy",
				Version: req.GetKubeproxyVersion(),
				Vulns:   convertNodeVulns(resp.GetKubeproxyVulnerabilities()),
			},
		},
	}
	if req.GetRuntime().GetName() != "" && req.GetRuntime().GetVersion() != "" {
		scan.Components = append(scan.Components, &storage.EmbeddedNodeScanComponent{
			Name:    req.GetRuntime().GetName(),
			Version: req.GetRuntime().GetVersion(),
			Vulns:   convertNodeVulns(resp.GetRuntimeVulnerabilities()),
		})
	}
	return scan
}

func convertNodeVulns(vulnerabilities []*v1.Vulnerability) []*storage.EmbeddedVulnerability {
	return convertVulnerabilities(vulnerabilities, storage.EmbeddedVulnerability_NODE_VULNERABILITY)
}

func convertK8sVulns(vulnerabilities []*v1.Vulnerability) []*storage.EmbeddedVulnerability {
	return convertVulnerabilities(vulnerabilities, storage.EmbeddedVulnerability_K8S_VULNERABILITY)
}

func convertVulnerabilities(vulnerabilities []*v1.Vulnerability, vulnType storage.EmbeddedVulnerability_VulnerabilityType) []*storage.EmbeddedVulnerability {
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(vulnerabilities))
	for _, vuln := range vulnerabilities {
		vulns = append(vulns, convertVulnerability(vuln, vulnType))
	}
	return vulns
}

// convertNodeVulnerability converts a clair vulnerability to a proto vulnerability
func convertVulnerability(v *v1.Vulnerability, vulnType storage.EmbeddedVulnerability_VulnerabilityType) *storage.EmbeddedVulnerability {
	link := v.GetLink()
	if link == "" {
		link = scans.GetVulnLink(v.GetName())
	}

	vuln := &storage.EmbeddedVulnerability{
		Cve:     v.GetName(),
		Summary: v.GetDescription(),
		Link:    link,
		SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: v.GetFixedBy(),
		},
		VulnerabilityType: vulnType,
		Severity:          clair.SeverityToStorageSeverity(v.GetSeverity()),
	}

	if v.GetMetadataV2() != nil {
		m := v.GetMetadataV2()

		vuln.PublishedOn = clair.ConvertTime(m.GetPublishedDateTime())
		vuln.LastModified = clair.ConvertTime(m.GetLastModifiedDateTime())
		if m.GetCvssV2() != nil && m.GetCvssV2().Vector != "" {
			if cvssV2, err := cvssv2.ParseCVSSV2(m.GetCvssV2().GetVector()); err == nil {
				cvssV2.ExploitabilityScore = m.GetCvssV2().GetExploitabilityScore()
				cvssV2.ImpactScore = m.GetCvssV2().GetImpactScore()
				cvssV2.Score = m.GetCvssV2().GetScore()

				vuln.CvssV2 = cvssV2
				// This sets the top level score for use in policies. It will be overwritten if v3 exists
				vuln.Cvss = cvssV2.GetScore()
				vuln.ScoreVersion = storage.EmbeddedVulnerability_V2
				vuln.CvssV2.Severity = cvssv2.Severity(vuln.GetCvss())
			} else {
				log.Errorf("converting Clairify CVSSv2: %v", err)
			}
		}

		if m.GetCvssV3() != nil && m.GetCvssV3().Vector != "" {
			if cvssV3, err := cvssv3.ParseCVSSV3(m.GetCvssV3().GetVector()); err == nil {
				cvssV3.ExploitabilityScore = m.GetCvssV3().GetExploitabilityScore()
				cvssV3.ImpactScore = m.GetCvssV3().GetImpactScore()
				cvssV3.Score = m.GetCvssV3().GetScore()

				vuln.CvssV3 = cvssV3

				vuln.Cvss = cvssV3.GetScore()
				vuln.ScoreVersion = storage.EmbeddedVulnerability_V3
				vuln.CvssV3.Severity = cvssv3.Severity(vuln.GetCvss())
			} else {
				log.Errorf("converting Clairify CVSSv3: %v", err)
			}
		}
	}

	return vuln
}

func convertImageToImageScan(metadata *storage.ImageMetadata, image *v1.Image) *storage.ImageScan {
	components := convertFeatures(metadata, image.GetFeatures())
	return &storage.ImageScan{
		ScanTime:        gogoProto.TimestampNow(),
		Components:      components,
		OperatingSystem: image.GetNamespace(),
	}
}

func convertFeatures(metadata *storage.ImageMetadata, features []*v1.Feature) []*storage.EmbeddedImageScanComponent {
	layerSHAToIndex := clair.BuildSHAToIndexMap(metadata)

	components := make([]*storage.EmbeddedImageScanComponent, 0, len(features))
	for _, feature := range features {
		convertedComponent := convertFeature(feature)
		if val, ok := layerSHAToIndex[feature.GetAddedByLayer()]; ok {
			convertedComponent.HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: val,
			}
		}
		components = append(components, convertedComponent)
	}

	return components
}

func convertFeature(feature *v1.Feature) *storage.EmbeddedImageScanComponent {
	component := &storage.EmbeddedImageScanComponent{
		Name:     feature.GetName(),
		Version:  feature.GetVersion(),
		Location: feature.GetLocation(),
		FixedBy:  feature.GetFixedBy(),
	}

	if source, ok := clair.VersionFormatsToSource[feature.GetFeatureType()]; ok {
		component.Source = source
	}
	component.Vulns = convertVulnerabilities(feature.GetVulnerabilities(), storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	executables := make([]*storage.EmbeddedImageScanComponent_Executable, 0, len(feature.GetProvidedExecutables()))
	for _, executable := range feature.GetProvidedExecutables() {
		imageComponentIds := make([]string, 0, len(executable.GetRequiredFeatures()))
		for _, f := range executable.GetRequiredFeatures() {
			imageComponentIds = append(imageComponentIds, scancomponent.ComponentID(f.GetName(), f.GetVersion(), ""))
		}
		exec := &storage.EmbeddedImageScanComponent_Executable{Path: executable.GetPath(), Dependencies: imageComponentIds}
		executables = append(executables, exec)
	}
	component.Executables = executables

	return component
}
