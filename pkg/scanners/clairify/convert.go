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

func convertNodeToVulnRequest(node *storage.Node) *v1.GetVulnerabilitiesRequest {
	// Components we support: kubelet, kube-proxy, container runtime, linux kernel
	components := make([]*v1.Component, 0, 4)

	if node.GetKubeletVersion() != "" {
		components = append(components, &v1.Component{
			Component: &v1.Component_K8SComponent{
				K8SComponent: &v1.KubernetesComponent{
					Component: v1.KubernetesComponent_KUBELET,
					Version:   node.GetKubeletVersion(),
				},
			},
		})
	}

	if node.GetKubeProxyVersion() != "" {
		components = append(components, &v1.Component{
			Component: &v1.Component_K8SComponent{
				K8SComponent: &v1.KubernetesComponent{
					Component: v1.KubernetesComponent_KUBE_PROXY,
					Version:   node.GetKubeProxyVersion(),
				},
			},
		})
	}

	containerRuntime := convertContainerRuntime(node.GetContainerRuntime())
	if containerRuntime != nil {
		components = append(components, &v1.Component{
			Component: &v1.Component_AppComponent{
				AppComponent: containerRuntime,
			},
		})
	}

	// Only linux is supported at this time.
	if node.GetOperatingSystem() == "linux" {
		components = append(components, &v1.Component{
			Component: &v1.Component_AppComponent{
				AppComponent: &v1.ApplicationComponent{
					Vendor:  "linux",
					Product: "linux_kernel",
					Version: node.GetKernelVersion(),
				},
			},
		})
	}

	return &v1.GetVulnerabilitiesRequest{
		Components: components,
	}
}

func convertContainerRuntime(containerRuntime *storage.ContainerRuntimeInfo) *v1.ApplicationComponent {
	var comp v1.ApplicationComponent
	switch containerRuntime.GetType() {
	case storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME:
		comp.Vendor = "docker"
		comp.Product = "docker"
		comp.Version = containerRuntime.GetVersion()
	case storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME:
		comp.Vendor = "kubernetes"
		comp.Product = "cri-o"
		comp.Version = containerRuntime.GetVersion()
	default:
		runtime, v := stringutils.Split2(containerRuntime.GetVersion(), "://")
		if runtime != "containerd" && runtime != "runc" {
			log.Warnf("Unsupported container runtime for node scanning: %s", runtime)
			return nil
		}
		comp.Vendor = "linuxfoundation"
		comp.Product = runtime
		comp.Version = v
	}

	return &comp
}

func convertVulnResponseToNodeScan(node *storage.Node, resp *v1.GetVulnerabilitiesResponse) *storage.NodeScan {
	componentsWithVulns := resp.GetVulnerabilitiesByComponent()
	if resp.GetVulnerabilitiesByComponent() == nil {
		return nil
	}

	components := make([]*storage.EmbeddedNodeScanComponent, 0, len(componentsWithVulns))
	for _, componentWithVulns := range componentsWithVulns {
		switch typ := componentWithVulns.GetComponent().GetComponent().(type) {
		case *v1.Component_AppComponent:
			component := typ.AppComponent
			name, v := getNameAndNodeVersion(node, component.GetProduct())
			if name == "" {
				continue
			}
			components = append(components, &storage.EmbeddedNodeScanComponent{
				Name:    name,
				Version: v,
				Vulns:   convertVulns(componentWithVulns.GetVulnerabilities()),
			})
		case *v1.Component_K8SComponent:
			component := typ.K8SComponent
			name, v := getNameAndNodeVersion(node, component.GetComponent().String())
			if name == "" {
				continue
			}
			components = append(components, &storage.EmbeddedNodeScanComponent{
				Name:    name,
				Version: v,
				Vulns:   convertVulns(componentWithVulns.GetVulnerabilities()),
			})
		default:
			log.Errorf("unsupported Node Component type %v", typ)
		}
	}

	return &storage.NodeScan{
		ScanTime:   gogoProto.TimestampNow(),
		Components: components,
	}
}

func getNameAndNodeVersion(node *storage.Node, component string) (string, string) {
	switch component {
	case v1.KubernetesComponent_KUBELET.String():
		return "kubelet", node.GetKubeletVersion()
	case v1.KubernetesComponent_KUBE_PROXY.String():
		return "kube-proxy", node.GetKubeProxyVersion()
	case "docker", "cri-o":
		return component, node.GetContainerRuntime().GetVersion()
	case "containerd", "runc":
		_, v := stringutils.Split2(node.GetContainerRuntime().GetVersion(), "://")
		return component, v
	case "linux_kernel":
		return "linux kernel", node.GetKernelVersion()
	default:
		return "", ""
	}
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
