package common

import (
	"sort"
	"strings"

	"github.com/stackrox/rox/image"
)

var (
	// ChartTemplates contains the list of currently supported chart names for helm related commands.
	ChartTemplates = map[string]image.ChartPrefix{
		ChartCentralServices:        image.CentralServicesChartPrefix,
		ChartSecuredClusterServices: image.SecuredClusterServicesChartPrefix,
		ChartOperator:               image.OperatorChartPrefix,
	}
	// PrettyChartNameList contains the list of currently supported chart names for Helm related
	// commands suitable for inline display.
	PrettyChartNameList string
)

const (
	// ChartCentralServices is the shortname for the StackRox Central Services Helm chart.
	ChartCentralServices string = "central-services"
	// ChartSecuredClusterServices is the shortname for the StackRox Secured Cluster Services Helm chart.
	ChartSecuredClusterServices string = "secured-cluster-services"
	// ChartOperator is the shortname for the StackRox Operator Helm chart.
	ChartOperator string = "operator"
)

// MakePrettyChartNameList forms a pretty printed string listing multiple chart names.
func MakePrettyChartNameList(chartNames ...string) string {
	sort.Strings(chartNames)
	return strings.Join(chartNames, " | ")
}

// Initialize `prettyChartNameList` for usage information.
func init() {
	chartTemplateNames := make([]string, 0, len(ChartTemplates))
	for name := range ChartTemplates {
		chartTemplateNames = append(chartTemplateNames, name)
	}
	PrettyChartNameList = MakePrettyChartNameList(chartTemplateNames...)
}
