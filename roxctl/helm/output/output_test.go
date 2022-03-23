package output

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestHelmOutputCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(helmOutputTestSuite))
}

type helmOutputTestSuite struct {
	suite.Suite
	helmOutputCommand helmOutputCommand
	errOur            *bytes.Buffer
}

func (suite *helmOutputTestSuite) SetupTest() {
	testIO, _, _, errOut := environment.TestIO()
	suite.helmOutputCommand = helmOutputCommand{}
	suite.helmOutputCommand.env = environment.NewCLIEnvironment(testIO, printer.DefaultColorPrinter())
	suite.errOur = errOut
}

func (suite *helmOutputTestSuite) TestInvalidCommandArgs() {
	cases := map[string]struct {
		args        []string
		shouldFail  bool
		error       error
		errorString string
	}{
		"should not return an error if valid number of arguments given with a correct chartName": {
			args: []string{common.ChartCentralServices},
		},
		"should return an error if no arguments given": {
			args:        []string{},
			shouldFail:  true,
			error:       errox.InvalidArgs,
			errorString: "incorrect number of arguments, see --help for usage information",
		},
		"should return an error if too many arguments given": {
			args:        []string{"test_chartName", "another_arg"},
			shouldFail:  true,
			error:       errox.InvalidArgs,
			errorString: "incorrect number of arguments, see --help for usage information",
		},
		"should return an error if invalid chart name given": {
			args:        []string{"invalid_chartName"},
			shouldFail:  true,
			error:       errox.InvalidArgs,
			errorString: "invalid arguments: unknown chart, see --help for list of supported chart names",
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			helmCmd := suite.helmOutputCommand
			cmd := Command(helmCmd.env)

			cmd.SetArgs(c.args)
			// Ignore an executing flow of the command
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				return nil
			}

			err := cmd.Execute()
			if c.shouldFail {
				suite.Require().Error(err)
				suite.Assert().ErrorIs(err, c.error)
				suite.Assert().Equal(c.errorString, err.Error())
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *helmOutputTestSuite) TestConstruct() {
	cmd := &cobra.Command{Use: "test"}

	cases := map[string]struct {
		chartName string
	}{
		"should not return an error if valid number of arguments given": {
			chartName: "test_chartName",
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			helmOutputCmd := suite.helmOutputCommand
			helmOutputCmd.Construct(c.chartName, cmd)
			suite.Assert().Equal(c.chartName, helmOutputCmd.chartName)
		})
	}

}

func (suite *helmOutputTestSuite) TestValidate() {
	cases := map[string]struct {
		chartName       string
		outputDir       string
		createOutputDir bool
		removeOutputDir bool
		errOutRegexp    string
		shouldFail      bool
		error           error
		errorRegexp     string
	}{
		"should not fail for valid chartName and provided outputDir": {
			chartName: common.ChartCentralServices,
			outputDir: "test_output_dir",
		},
		"should not fail for valid chartName and non provided outputDir": {
			chartName:    common.ChartCentralServices,
			errOutRegexp: "WARN:\tNo output directory specified, using default directory \"./stackrox-central-services-chart\"",
		},
		"should not fail for valid chartName and existed outputDir": {
			chartName:       common.ChartCentralServices,
			createOutputDir: true,
			removeOutputDir: true,
			errOutRegexp:    "WARN:\tRemoved output directory .*",
		},
		"should fail for already existed output directory": {
			chartName:       common.ChartCentralServices,
			createOutputDir: true,
			removeOutputDir: false,
			errOutRegexp:    "ERROR:\tDirectory .* already exists, use --remove or select a different directory with --output-dir.",
			shouldFail:      true,
			error:           errox.AlreadyExists,
			errorRegexp:     "directory \".*\" already exists",
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			helmOutputCmd := suite.helmOutputCommand
			helmOutputCmd.chartName = c.chartName
			helmOutputCmd.removeOutputDir = c.removeOutputDir
			helmOutputCmd.outputDir = c.outputDir
			if c.createOutputDir {
				outputDir, mkDirErr := os.MkdirTemp("", "roxctl-helm-output-")
				suite.T().Cleanup(func() {
					_ = os.RemoveAll(outputDir)
				})
				require.NoError(suite.T(), mkDirErr)
				helmOutputCmd.outputDir = outputDir
			}

			err := helmOutputCmd.Validate()
			suite.Assert().Regexp(c.errOutRegexp, suite.errOur.String())
			if c.shouldFail {
				suite.Require().Error(err)
				suite.Assert().ErrorIs(err, c.error)
				suite.Assert().Regexp(c.errorRegexp, err.Error())
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
