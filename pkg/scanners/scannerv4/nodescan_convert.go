package scannerv4

import (
	"fmt"
	"regexp"
	"strings"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
)

const rhcosFullName = "Red Hat Enterprise Linux CoreOS"

var rhcosOSImagePattern = regexp.MustCompile(`(Red Hat Enterprise Linux) (CoreOS) (\d)(\d+)`)

func nodeScan(osImage string, vr *v4.VulnerabilityReport) *storage.NodeScan {
	return &storage.NodeScan{
		ScannerVersion:  storage.NodeScan_SCANNER_V4,
		ScanTime:        protocompat.TimestampNow(),
		OperatingSystem: nodeOS(osImage),
		Components:      nodeComponents(vr),
		Notes:           nodeNotes(vr, osImage),
	}
}

func nodeOS(osImage string) string {
	r := rhcosOSImagePattern.FindStringSubmatch(osImage)
	if len(r) != 5 {
		return "unknown"
	}
	return fmt.Sprintf("rhcos:%s.%s", r[3], r[4])
}

func nodeComponents(report *v4.VulnerabilityReport) []*storage.EmbeddedNodeScanComponent {
	pkgs := report.GetContents().GetPackages()
	components := make([]*storage.EmbeddedNodeScanComponent, 0, len(pkgs))
	for _, pkg := range pkgs {
		id := pkg.GetId()
		vulnIDs := report.GetPackageVulnerabilities()[id].GetValues()

		component := &storage.EmbeddedNodeScanComponent{
			Name:    pkg.GetName(),
			Version: pkg.GetVersion(),
			Vulns:   vulnerabilities(vulnIDs, report.GetVulnerabilities(), storage.EmbeddedVulnerability_NODE_VULNERABILITY),
		}

		components = append(components, component)
	}
	return components
}

func nodeNotes(report *v4.VulnerabilityReport, osImage string) []storage.NodeScan_Note {
	notes := set.NewSet[storage.NodeScan_Note]()
	for _, note := range report.GetNotes() {
		switch note {
		case v4.VulnerabilityReport_NOTE_UNSPECIFIED:
			notes.Add(storage.NodeScan_UNSET)
		case v4.VulnerabilityReport_NOTE_OS_UNKNOWN, v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED:
			notes.Add(storage.NodeScan_UNSUPPORTED)
		default:
			// Ignore unknown/unsupported note.
			log.Warnf("encountered unknown node note (%v); skipping...", note)
		}
	}
	return filterNodeNotes(notes.AsSlice(), osImage)
}

// TODO(ROX-26593): Instead of fixing notes here, add RHCOS DistributionScanner to Claircore.
// All nodes currently get the note UNSUPPORTED assigned to them because the IndexReport does not contain
// Distribution information. To include it there, a specialized RHCOS DistributionScanner needs to be added
// to Claircore and then called in Compliances' IndexNode function where the IndexReport is created.
func filterNodeNotes(notes []storage.NodeScan_Note, osImage string) []storage.NodeScan_Note {
	if !strings.HasPrefix(osImage, rhcosFullName) {
		// Keep notes as they are for nodes other than RHCOS.
		return notes
	}
	filtered := notes[:0]
	for _, note := range notes {
		if note == storage.NodeScan_UNSUPPORTED {
			continue
		}
		filtered = append(filtered, note)
	}
	return filtered
}
