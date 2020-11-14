package scan

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gookit/color"
	"github.com/mitchellh/go-wordwrap"
	"github.com/stackrox/rox/generated/storage"
)

/**
 * Print scan result in a human readable format as follows:
 * Layer: ADD file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in /
 *   apt 1.0.9.8.4
 *     CVE-2019-3462 (CVSS 8.2) - fixed by 1.0.9.8.5
 *       * Incorrect sanitation of the 302 redirect field in HTTP transport method of apt versions 1.4.8 and earlier can lead to content injection by a MITM attacker, potentially leading to remote code execution on the target machine.
 *     CVE-2011-3374 (CVSS 3.7)
 *       * It was found that apt-key in apt, all versions, do not correctly validate gpg keys with the master keyring, leading to a potential man-in-the- middle attack
 */

// Output formats
const (
	layerFormat     = "Layer: %s %s\n"
	componentFormat = "  %s %s\n"
	vulnFormat      = "    %s (CVSS %g)%s\n"
	summaryFormat   = "      * %s\n"
	fixedByFormat   = " - fixed by %s"
)

// PrintPretty prints image scan result in a human readable format
func PrintPretty(imageResult *storage.Image) {
	// Sort components by layerIndex
	components := imageResult.GetScan().GetComponents()
	sort.SliceStable(components, func(p, q int) bool {
		return components[p].GetLayerIndex() < components[q].GetLayerIndex() || components[p].GetLayerIndex() == components[q].GetLayerIndex() && components[p].GetName() < components[q].GetName()
	})

	var layerHeader string
	var ci int
	noColorEnv := os.Getenv("NO_COLOR")
	noColor := noColorEnv != "0" && noColorEnv != ""
	for li, layer := range imageResult.GetMetadata().GetV1().GetLayers() {
		layerHeader = fmt.Sprintf(layerFormat, layer.Instruction, layer.Value)
		for ci < len(components) && components[ci].GetLayerIndex() == int32(li) {
			component := components[ci]
			vulns := component.GetVulns()
			ci++
			if len(vulns) == 0 {
				continue
			}
			if layerHeader != "" {
				fmt.Print(layerHeader)
				layerHeader = ""
			}
			if noColor {
				color.Printf(componentFormat, component.GetName(), component.GetVersion())
			} else {
				color.Bold.Printf(componentFormat, component.GetName(), component.GetVersion())
			}
			// Sort vulns reversely by CVSS score
			sort.SliceStable(vulns, func(p, q int) bool { return vulns[p].GetCvss() > vulns[q].GetCvss() })
			for _, vuln := range vulns {
				var colorPrint func(format string, a ...interface{})
				switch {
				case vuln.Cvss < 4 || noColor:
					colorPrint = color.Printf
				case vuln.Cvss >= 7:
					colorPrint = color.Danger.Printf
				default:
					colorPrint = color.Warn.Printf
				}

				if vuln.GetFixedBy() == "" {
					colorPrint(vulnFormat, vuln.GetCve(), vuln.GetCvss(), "")
				} else {
					fixedBy := fmt.Sprintf(fixedByFormat, vuln.GetFixedBy())
					colorPrint(vulnFormat, vuln.GetCve(), vuln.GetCvss(), fixedBy)
				}
				wrapped := wordwrap.WrapString(vuln.GetSummary(), 120)
				wrapped = strings.TrimSpace(wrapped)
				wrapped = strings.Replace(wrapped, "\n", "\n        ", -1)
				fmt.Printf(summaryFormat, wrapped)
			}
			fmt.Println("")
		}
	}
}
