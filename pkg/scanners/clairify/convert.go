package clairify

import (
	"time"

	gogoProto "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cvss/cvssv2"
	"github.com/stackrox/rox/pkg/cvss/cvssv3"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/scans"
	"github.com/stackrox/rox/pkg/stringutils"
	v1 "github.com/stackrox/scanner/generated/shared/api/v1"
)

const (
	timeFormat = "2006-01-02T15:04Z"
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
		ScanTime: gogoProto.TimestampNow(),
		Components: []*storage.EmbeddedNodeScanComponent{
			{
				Name:    "kernel",
				Version: req.GetKernelVersion(),
				Vulns:   convertVulns(resp.GetKernelVulnerabilities()),
			},
			{
				Name:    "kubelet",
				Version: req.GetKubeletVersion(),
				Vulns:   convertVulns(resp.GetKubeletVulnerabilities()),
			},
			{
				Name:    "kube-proxy",
				Version: req.GetKubeproxyVersion(),
				Vulns:   convertVulns(resp.GetKubeproxyVulnerabilities()),
			},
		},
	}
	if req.GetRuntime().GetName() != "" && req.GetRuntime().GetVersion() != "" {
		scan.Components = append(scan.Components, &storage.EmbeddedNodeScanComponent{
			Name:    req.GetRuntime().GetName(),
			Version: req.GetRuntime().GetVersion(),
			Vulns:   convertVulns(resp.GetRuntimeVulnerabilities()),
		})
	}
	return scan
}

func convertVulns(vulnerabilities []*v1.Vulnerability) []*storage.EmbeddedVulnerability {
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(vulnerabilities))
	for _, vuln := range vulnerabilities {
		vulns = append(vulns, convertNodeVulnerability(vuln))
	}
	return vulns
}

// convertNodeVulnerability converts a clair node vulnerability to a proto vulnerability
func convertNodeVulnerability(v *v1.Vulnerability) *storage.EmbeddedVulnerability {
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
		VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
	}

	if v.GetMetadataV2() != nil {
		m := v.GetMetadataV2()
		if m.GetPublishedDateTime() != "" {
			if ts, err := time.Parse(timeFormat, m.GetPublishedDateTime()); err == nil {
				vuln.PublishedOn = protoconv.ConvertTimeToTimestamp(ts)
			}
		}
		if m.GetLastModifiedDateTime() != "" {
			if ts, err := time.Parse(timeFormat, m.GetLastModifiedDateTime()); err == nil {
				vuln.LastModified = protoconv.ConvertTimeToTimestamp(ts)
			}
		}

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
