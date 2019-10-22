package vulns

import (
	"math"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mathutil"
)

// ProcessComponents takes in a slice of components and outputs the min, max, sum CVSS scores as well as the number of CVEs
func ProcessComponents(components []*storage.EmbeddedImageScanComponent) (min, max, sum float32, num int) {
	min = math.MaxFloat32
	max = -math.MaxFloat32
	for _, component := range components {
		cMin, cMax, cSum, cNum := ProcessComponent(component)
		if cNum == 0 {
			continue
		}

		min = mathutil.MinFloat32(cMin, min)
		max = mathutil.MaxFloat32(cMax, max)
		sum += cSum
		num += cNum
	}
	return min, max, sum, num
}

// ProcessComponent takes in a single component and outputs the min, max, sum CVSS scores as well as the number of CVEs
func ProcessComponent(component *storage.EmbeddedImageScanComponent) (min, max, sum float32, numCVEs int) {
	min = math.MaxFloat32
	max = -math.MaxFloat32
	for _, vuln := range component.GetVulns() {
		// Sometimes if the vuln doesn't have a CVSS score then it is unknown and we'll exclude it during scoring
		if vuln.GetCvss() == 0 || !strings.HasPrefix(vuln.GetCve(), "CVE") {
			continue
		}
		max = mathutil.MaxFloat32(vuln.GetCvss(), max)
		min = mathutil.MinFloat32(vuln.GetCvss(), min)
		sum += vuln.GetCvss() * vuln.GetCvss() / 10
		numCVEs++
	}
	if numCVEs == 0 {
		return 0, 0, 0, 0
	}
	return min, max, sum, numCVEs
}
