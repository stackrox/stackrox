package derivelocalvalues

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/suite"
)

func TestHelmDeriveLocalValuesCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(helmDeriveLocalValuesTestSuite))
}

type helmDeriveLocalValuesTestSuite struct {
	suite.Suite
	helmDeriveLocalValuesCommand helmDeriveLocalValuesCommand
}

func (suite *helmDeriveLocalValuesTestSuite) SetupTest() {
	testIO, _, _, _ := environment.TestIO()
	suite.helmDeriveLocalValuesCommand = helmDeriveLocalValuesCommand{}
	suite.helmDeriveLocalValuesCommand.env = environment.NewCLIEnvironment(testIO, printer.DefaultColorPrinter())
}

func (suite *helmDeriveLocalValuesTestSuite) TestConstruct() {
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
			helmCmd := suite.helmDeriveLocalValuesCommand
			err := helmCmd.Construct(c.args, cmd)
			if c.shouldFail {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
			suite.Assert().Equal(c.chartName, helmCmd.chartName)
		})
	}
}

func (suite *helmDeriveLocalValuesTestSuite) TestValidate() {
	cases := map[string]struct {
		output       string
		outputDir    string
		outputPath   string
		useDirectory bool
		shouldFail   bool
		error        error
	}{
		"should not fail for valid output argument and empty outputDir": {
			output:     "path_to_file",
			outputPath: "path_to_file",
		},
		"should not fail for empty output argument and valid outputDir": {
			outputDir:    "path_to_folder",
			outputPath:   "path_to_folder",
			useDirectory: true,
		},
		"should use standardOutput if output is equal to '-'": {
			output:     "-",
			outputPath: standardOutput,
		},
		"should fail if both output and outputDir are empty": {
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
		"should fail if both output and outputDir given": {
			output:     "path_to_file",
			outputDir:  "path_to_folder",
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			helmCmd := suite.helmDeriveLocalValuesCommand
			helmCmd.logger = helmCmd.env.Logger()
			helmCmd.output = c.output
			helmCmd.outputDir = c.outputDir

			err := helmCmd.Validate()
			if c.shouldFail {
				suite.Require().Error(err)
				suite.Assert().ErrorIs(err, c.error)
			} else {
				suite.Require().NoError(err)
			}
			suite.Assert().Equal(c.outputPath, helmCmd.outputPath)
			suite.Assert().Equal(c.useDirectory, helmCmd.useDirectory)
		})
	}
}
