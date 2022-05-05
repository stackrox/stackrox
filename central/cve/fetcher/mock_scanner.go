package fetcher

import (
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/types"
)

// nvdCVEWithComponents is to simulate the k8s vulnerabilities in scanner.
// Central does not differentiate the components in vulnerabilities in scan.
type nvdCVEWithComponents struct {
	nvdCVE     *schema.NVDCVEFeedJSON10DefCVEItem
	components []string
}

type mockScanner struct {
	types.ScanSemaphore
	cveMatcher *matcher.CVEMatcher
	nvdCVEs    []*nvdCVEWithComponents
}

func (o *mockScanner) Name() string {
	return "mockOrchestratorScanner1"
}

func (o *mockScanner) Type() string {
	return "mockOrchestratorScanner"
}

func (o *mockScanner) KubernetesScan(version string) (map[string][]*storage.EmbeddedVulnerability, error) {
	vulnsMap := make(map[string][]*storage.EmbeddedVulnerability)
	for _, cve := range o.nvdCVEs {
		if len(cve.components) == 1 && cve.components[0] == "openshift" {
			continue
		}
		for _, node := range cve.nvdCVE.Configurations.Nodes {
			embeddedCve, err := converter.NvdCVEToEmbeddedCVE(cve.nvdCVE, converter.K8s)
			if err != nil {
				return nil, err
			}
			matched, err := o.cveMatcher.MatchVersions(node, version, converter.K8s)
			if err != nil {
				return nil, err
			}
			if matched {
				for _, component := range cve.components {
					if _, exists := vulnsMap[component]; !exists {
						vulnsMap[component] = make([]*storage.EmbeddedVulnerability, 0)
					}
					vulnsMap[component] = append(vulnsMap[component], embeddedCve)
				}
				break
			}
		}
	}
	return vulnsMap, nil
}

func (o *mockScanner) OpenShiftScan(version string) ([]*storage.EmbeddedVulnerability, error) {
	var vulns []*storage.EmbeddedVulnerability
	for _, cve := range o.nvdCVEs {
		if len(cve.components) != 1 || cve.components[0] != "openshift" {
			continue
		}
		for _, node := range cve.nvdCVE.Configurations.Nodes {
			embeddedCve, err := converter.NvdCVEToEmbeddedCVE(cve.nvdCVE, converter.OpenShift)
			if err != nil {
				return nil, err
			}
			matched, err := o.cveMatcher.MatchVersions(node, version, converter.OpenShift)
			if err != nil {
				return nil, err
			}
			if matched {
				vulns = append(vulns, embeddedCve)
				break
			}
		}
	}
	return vulns, nil
}
