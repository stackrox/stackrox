package nvd

import (
	"strings"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	cpeVersionIndex = 5
	cpeUpdateIndex  = 6
	cpePartsLength  = 13

	// Any represent CPE Any string punctuation
	Any = "*"
	// NotApplicable represents N/A CPE string punctuation
	NotApplicable = "-"
)

// FillMissingData fills NVD CVEs with missing data, e.g. affected product versions
func FillMissingData(project Project, cves []*schema.NVDCVEFeedJSON10DefCVEItem) []*schema.NVDCVEFeedJSON10DefCVEItem {
	cves = addMissingCVEs(project, cves)
	tryUpdateCVEWithAffectedVersions(cves)
	return cves
}

func addMissingCVEs(project Project, cves []*schema.NVDCVEFeedJSON10DefCVEItem) []*schema.NVDCVEFeedJSON10DefCVEItem {
	cvesFromNVD := make(map[string]struct{})
	for _, cve := range cves {
		cvesFromNVD[cve.CVE.CVEDataMeta.ID] = struct{}{}
	}

	for _, cve := range cvesNotInDataFeed[project] {
		if _, ok := cvesFromNVD[cve.CVE.CVEDataMeta.ID]; !ok {
			cves = append(cves, cve)
		}
	}
	return cves
}

func tryUpdateCVEWithAffectedVersions(cves []*schema.NVDCVEFeedJSON10DefCVEItem) {
	for _, cve := range cves {
		if _, ok := cvesWithoutFixedBy[cve.CVE.CVEDataMeta.ID]; !ok {
			continue
		}

		updateRequired := true
		filteredNodes := make([]*schema.NVDCVEFeedJSON10DefNode, 0, len(cve.Configurations.Nodes))
		for _, node := range cve.Configurations.Nodes {
			filteredCPE := make([]*schema.NVDCVEFeedJSON10DefCPEMatch, 0, len(node.CPEMatch))
			for _, cpeMatch := range node.CPEMatch {
				hasVersionInfo, err := cpeHasVersionInfo(cpeMatch)
				if err != nil {
					log.Error(err)
					continue
				}

				if !hasVersionInfo {
					continue
				}

				updateRequired = false
				// We filter because we want to get rid of any CPE not providing applicability statement.
				filteredCPE = append(filteredCPE, cpeMatch)
			}

			if len(filteredCPE) > 0 {
				filteredNodes = append(filteredNodes, &schema.NVDCVEFeedJSON10DefNode{
					CPEMatch: filteredCPE,
					Operator: node.Operator,
					Children: node.Children,
					Negate:   node.Negate,
				})
			}
		}

		// For the concerned CVEs, if NVD is providing data, use it, else update CVE.
		cve.Configurations.Nodes = filteredNodes
		if updateRequired {
			cve.Configurations.Nodes = append(cve.Configurations.Nodes, cvesWithoutFixedBy[cve.CVE.CVEDataMeta.ID]...)
		}
	}
}

func getVersionAndUpdate(cpe string) (string, string, error) {
	if ok := strings.HasPrefix(cpe, "cpe:2.3:a:"); !ok {
		return "", "", errors.Errorf("cpe: %q not a valid cpe23Uri format", cpe)
	}

	ss := strings.Split(cpe, ":")
	if len(ss) != cpePartsLength {
		return "", "", errors.Errorf("cpe: %q not a valid cpe23Uri format", cpe)
	}
	return ss[cpeVersionIndex], ss[cpeUpdateIndex], nil
}

func cpeHasVersionInfo(cpeMatch *schema.NVDCVEFeedJSON10DefCPEMatch) (bool, error) {
	version, update, err := getVersionAndUpdate(cpeMatch.Cpe23Uri)
	if err != nil {
		return false, errors.Wrapf(err, "could not get version and update from cpe: %q", cpeMatch.Cpe23Uri)
	}

	// Possibly invalid CPE
	if version == "" && update == "" {
		return false, nil
	}

	// CPE string does not provide affected versions information
	if version == NotApplicable {
		return false, nil
	}

	// If CPE string matches any version check if ranges are provided. Note: CPE23URI provides affected versions and not ranges.
	if version == Any && update == Any {
		return cpeMatch.VersionStartIncluding != "" || cpeMatch.VersionEndIncluding != "" || cpeMatch.VersionEndExcluding != "", nil
	}

	return true, nil
}
