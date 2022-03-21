package output

import (
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
}

func (suite *helmOutputTestSuite) SetupTest() {
	testIO, _, _, _ := environment.TestIO()
	suite.helmOutputCommand = helmOutputCommand{}
	suite.helmOutputCommand.env = environment.NewCLIEnvironment(testIO, printer.DefaultColorPrinter())
}

func (suite *helmOutputTestSuite) TestConstruct() {
	cmd := &cobra.Command{Use: "test"}

	cases := map[string]struct {
		args       []string
		chartName  string
		shouldFail bool
	}{
		"should not return an error if valid number of arguments given": {
			args:      []string{"test_chartName"},
			chartName: "test_chartName",
		},
		"should return an error if no arguments given": {
			args:       []string{},
			shouldFail: true,
		},
		"should return an error if too many arguments given": {
			args:       []string{"test_chartName", "another_arg"},
			shouldFail: true,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			helmOutputCmd := suite.helmOutputCommand
			err := helmOutputCmd.Construct(c.args, cmd)
			if c.shouldFail {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
			suite.Assert().Equal(c.chartName, helmOutputCmd.chartName)
		})
	}

}

func (suite *helmOutputTestSuite) TestValidate() {
	cases := map[string]struct {
		chartName       string
		createOutputDir bool
		removeOutputDir bool
		shouldFail      bool
		error           error
	}{
		"should not fail for valid chartName and non provided outputDir": {
			chartName: common.ChartCentralServices,
		},
		"should not fail for valid chartName and existed outputDir": {
			chartName:       common.ChartCentralServices,
			createOutputDir: true,
			removeOutputDir: true,
		},
		"should fail for already existed output directory": {
			chartName:       common.ChartCentralServices,
			createOutputDir: true,
			removeOutputDir: false,
			shouldFail:      true,
			error:           errox.AlreadyExists,
		},
		"should fail for incorrect chartName": {
			chartName:  "wrong_chart_name",
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			helmOutputCmd := suite.helmOutputCommand
			helmOutputCmd.logger = helmOutputCmd.env.Logger()
			helmOutputCmd.chartName = c.chartName
			helmOutputCmd.removeOutputDir = c.removeOutputDir
			if c.createOutputDir {
				outputDir, mkDirErr := os.MkdirTemp("", "roxctl-helm-output-")
				suite.T().Cleanup(func() {
					_ = os.RemoveAll(outputDir)
				})
				require.NoError(suite.T(), mkDirErr)
				helmOutputCmd.outputDir = outputDir
			}

			err := helmOutputCmd.Validate()
			if c.shouldFail {
				suite.Require().Error(err)
				suite.Assert().ErrorIs(err, c.error)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
