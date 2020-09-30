package common

import (
	"sort"
	"strings"

	"github.com/stackrox/rox/image"
)

var (
	// ChartTemplates contains the list of currently supported chart names for helm related commands.
	ChartTemplates = map[string]string{
		ChartCentralServices: image.CentralServicesChartPrefix,
	}
	// PrettyChartNameList contains the list of currently supported chart names for helm relateld
	// commands suitable for inline display.
	PrettyChartNameList string
)

const (
	// ChartCentralServices is the shortname for the StackRox Central Services Helm chart.
	ChartCentralServices string = "central-services"
)

// Initialize `prettyChartNameList` for usage information.
func init() {
	chartTemplateNames := make([]string, 0, len(ChartTemplates))
	for name := range ChartTemplates {
		chartTemplateNames = append(chartTemplateNames, name)
	}
	sort.Strings(chartTemplateNames)
	PrettyChartNameList = strings.Join(chartTemplateNames, " | ")
}
