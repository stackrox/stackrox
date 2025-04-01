package export

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/central/export/deployments"
	"github.com/stackrox/rox/roxctl/central/export/images"
	"github.com/stackrox/rox/roxctl/central/export/nodes"
	"github.com/stackrox/rox/roxctl/central/export/pods"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	defaultExportTimeout = 10 * time.Minute
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "export",
		Short: "(Technology Preview) Commands related to exporting data from Central",
		Long:  "Commands related to exporting data from Central." + common.TechPreviewLongText,
	}
	c.AddCommand(
		deployments.Command(cliEnvironment),
		images.Command(cliEnvironment),
		nodes.Command(cliEnvironment),
		pods.Command(cliEnvironment),
	)
	flags.AddTimeoutWithDefault(c, defaultExportTimeout)
	return c
}
