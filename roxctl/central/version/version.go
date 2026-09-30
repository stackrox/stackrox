package version

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stackrox/rox/pkg/version/versioncompatibility"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"github.com/stackrox/rox/roxctl/common/versioncheck"
)

type centralVersionCommand struct {
	env          environment.Environment
	timeout      time.Duration
	retryTimeout time.Duration
}

type versionResult struct {
	RoxctlVersion             string   `json:"RoxctlVersion"`
	CentralVersion            string   `json:"CentralVersion"`
	CompatibleCentralVersions []string `json:"CompatibleCentralVersions"`
	Compatibility             string   `json:"Compatibility"`
	Guidance                  string   `json:"Guidance"`

	compatibility versioncompatibility.Compatibility
}

// Command defines the central version command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cbr := &cobra.Command{
		Use:           "version",
		Short:         "Display Central's version and check compatibility with this roxctl",
		SilenceErrors: true,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cmd := makeCentralVersionCommand(cliEnvironment, c)
			useJSON, _ := c.Flags().GetBool("json")
			return cmd.run(useJSON)
		}),
	}

	flags.AddTimeout(cbr)
	flags.AddRetryTimeout(cbr)
	cbr.PersistentFlags().Bool("json", false, "Display version and compatibility information as JSON.")
	return cbr
}

func makeCentralVersionCommand(cliEnvironment environment.Environment, cbr *cobra.Command) *centralVersionCommand {
	return &centralVersionCommand{
		env:          cliEnvironment,
		timeout:      flags.Timeout(cbr),
		retryTimeout: flags.RetryTimeout(cbr),
	}
}

func (cmd *centralVersionCommand) run(useJSON bool) error {
	result, err := cmd.fetchAndClassify()
	if err != nil {
		cmd.env.Logger().ErrfLn("%v", err)
		return err
	}

	if useJSON {
		if err := cmd.printJSON(result); err != nil {
			cmd.env.Logger().ErrfLn("%v", err)
			return err
		}
		return nil
	}
	cmd.printText(result)
	return nil
}

func (cmd *centralVersionCommand) fetchAndClassify() (*versionResult, error) {
	roxctlVersion := version.GetMainVersion()

	conn, err := cmd.env.GRPCConnection(common.WithRetryTimeout(cmd.retryTimeout))
	if err != nil {
		return nil, errors.Wrap(err, "establishing gRPC connection to Central")
	}
	defer utils.IgnoreError(conn.Close)

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	versioncheck.SuppressWarning() // Don't print another warning if versions

	metadata, err := v1.NewMetadataServiceClient(conn).GetMetadata(ctx, &v1.Empty{})
	if err != nil {
		return nil, errors.Wrap(err, "getting Central metadata")
	}

	centralVersion := metadata.GetVersion()
	if centralVersion == "" {
		return nil, errors.New(
			"Central did not return its version. " +
				"This typically means roxctl is not authenticated. " +
				"Run \"roxctl central login\" first")
	}

	centralXY, err := productstreams.ParseXYFromVersionString(centralVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing Central version %q", centralVersion)
	}

	compat, err := versioncompatibility.ClassifyVersion(centralXY)
	if err != nil {
		return nil, errors.Wrap(err, "classifying Central version")
	}

	compatVersions, err := versioncompatibility.CompatibleVersions()
	if err != nil {
		return nil, errors.Wrap(err, "getting compatible versions")
	}
	compatStrs := make([]string, 0, len(compatVersions))
	for _, v := range compatVersions {
		compatStrs = append(compatStrs, v.String())
	}

	return &versionResult{
		RoxctlVersion:             roxctlVersion,
		CentralVersion:            centralVersion,
		CompatibleCentralVersions: compatStrs,
		Compatibility:             compat.String(),
		Guidance:                  versioncheck.Guidance(compat).String(),
		compatibility:             compat,
	}, nil
}

func (cmd *centralVersionCommand) printText(r *versionResult) {
	const labelFmt = "%-35s%s"
	cmd.env.Logger().PrintfLn(labelFmt, "Central version:", r.CentralVersion)
	cmd.env.Logger().PrintfLn(labelFmt, "roxctl version:", r.RoxctlVersion)
	cmd.env.Logger().PrintfLn(labelFmt, "  Compatible Central versions:", strings.Join(r.CompatibleCentralVersions, ", "))
	cmd.env.Logger().PrintfLn(labelFmt, "Compatibility:", r.compatibility.DisplayName())
	g := versioncheck.Guidance(r.compatibility)
	cmd.env.Logger().PrintfLn("  %s", g.Summary)
	if g.Recommendation != "" {
		cmd.env.Logger().PrintfLn("  %s", g.Recommendation)
	}
}

func (cmd *centralVersionCommand) printJSON(r *versionResult) error {
	enc := json.NewEncoder(cmd.env.InputOutput().Out())
	enc.SetIndent("", "  ")
	return errors.Wrap(enc.Encode(r), "encoding version information as JSON")
}
