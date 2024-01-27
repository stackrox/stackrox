package output

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/buildinfo"
	pkgEnv "github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/version/testutils"
	env "github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"github.com/stretchr/testify/assert"
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
	testIO, _, _, errOut := io.TestIO()
	suite.helmOutputCommand = helmOutputCommand{}
	suite.helmOutputCommand.env = env.NewTestCLIEnvironment(suite.T(), testIO, printer.DefaultColorPrinter())
	suite.errOur = errOut
}

func TestTelemetryConfiguration(t *testing.T) {
	dirtyVersion := "1.2.3-dirty"
	releaseVersion := "1.2.3"
	var disabledInDebug any
	if !buildinfo.ReleaseBuild || buildinfo.TestBuild {
		disabledInDebug = phonehome.DisabledKey
	}

	testIO, _, _, _ := io.TestIO()
	cmd := helmOutputCommand{
		imageFlavor: "opensource",
		env:         env.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter()),
	}

	type result struct {
		enabled bool
		key     interface{}
	}

	disabled := result{enabled: false, key: phonehome.DisabledKey}

	testCases := []struct {
		testName  string
		version   string
		telemetry bool
		key       string
		expected  result
	}{
		{testName: "test1", version: dirtyVersion, telemetry: true, key: "", expected: disabled},
		{testName: "test2", version: dirtyVersion, telemetry: false, key: "", expected: disabled},
		{testName: "test3", version: dirtyVersion, telemetry: true, key: "KEY", expected: result{enabled: true, key: "KEY"}},
		{testName: "test4", version: dirtyVersion, telemetry: false, key: "KEY", expected: disabled},

		{testName: "test5", version: releaseVersion, telemetry: true, key: "", expected: result{enabled: buildinfo.ReleaseBuild && !buildinfo.TestBuild, key: disabledInDebug}},
		{testName: "test6", version: releaseVersion, telemetry: false, key: "", expected: disabled},
		{testName: "test7", version: releaseVersion, telemetry: true, key: "KEY", expected: result{enabled: true, key: "KEY"}},
		{testName: "test8", version: releaseVersion, telemetry: false, key: "KEY", expected: disabled},

		{testName: "test9", version: dirtyVersion, telemetry: true, key: phonehome.DisabledKey, expected: disabled},
		{testName: "test10", version: dirtyVersion, telemetry: false, key: phonehome.DisabledKey, expected: disabled},
		{testName: "test11", version: releaseVersion, telemetry: true, key: phonehome.DisabledKey, expected: disabled},
		{testName: "test12", version: releaseVersion, telemetry: false, key: phonehome.DisabledKey, expected: disabled},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			t.Setenv(pkgEnv.TelemetryStorageKey.EnvVar(), testCase.key)
			testutils.SetMainVersion(t, testCase.version)

			cmd.telemetry = testCase.telemetry
			values, err := cmd.getChartMetaValues(true)
			require.NoError(t, err, err)
			assert.Equal(t, testCase.expected.enabled, values.TelemetryEnabled)
			assert.Equal(t, testCase.expected.key, values.TelemetryKey)
		})
	}
}

func (suite *helmOutputTestSuite) TestInvalidCommandArgs() {
	cases := map[string]struct {
		args       []string
		shouldFail bool
		errStdout  string
	}{
		"should not return an error if valid number of arguments given with a correct chartName": {
			args: []string{common.ChartCentralServices},
		},
		"should return an error if no arguments given": {
			args:       []string{},
			shouldFail: true,
			errStdout:  "Error: accepts 1 arg(s), received 0\n",
		},
		"should return an error if too many arguments given": {
			args:       []string{"test_chartName", "another_arg"},
			shouldFail: true,
			errStdout:  "Error: accepts 1 arg(s), received 2\n",
		},
		"should return an error if invalid chart name given": {
			args:       []string{"invalid_chartName"},
			shouldFail: true,
			errStdout: `Error: invalid argument "invalid_chartName" for "output"
`,
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
			// Redirect stdErr
			errOut := bytes.NewBufferString("")
			cmd.SetErr(errOut)

			err := cmd.Execute()
			if c.shouldFail {
				suite.Require().Error(err)
				suite.Assert().Equal(c.errStdout, errOut.String())
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *helmOutputTestSuite) TestConstruct() {
	cmd := &cobra.Command{Use: "test"}
	chartName := "test_chartName"

	helmOutputCmd := suite.helmOutputCommand
	helmOutputCmd.Construct(chartName, cmd)
	suite.Assert().Equal(chartName, helmOutputCmd.chartName)
}

func (suite *helmOutputTestSuite) TestValidate() {
	cases := map[string]struct {
		chartName    string
		outputDir    string
		createOutDir bool
		removeOutDir bool
		errOutRegexp string
		shouldFail   bool
		error        error
		errorRegexp  string
	}{
		"should not fail for valid chartName and provided outputDir": {
			outputDir: "test_output_dir",
		},
		"should not fail for valid chartName and non provided outputDir": {
			errOutRegexp: `WARN:	No output directory specified, using default directory "./stackrox-central-services-chart"`,
		},
		"should not fail for valid chartName and existed outputDir": {
			createOutDir: true,
			removeOutDir: true,
			errOutRegexp: "WARN:	Removed output directory .*",
		},
		"should fail for already existed output directory": {
			error:        errox.AlreadyExists,
			shouldFail:   true,
			errorRegexp:  `directory ".*" already exists`,
			createOutDir: true,
			removeOutDir: false,
			errOutRegexp: "ERROR:	Directory .* already exists, use --remove or select a different directory with --output-dir.",
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			helmOutputCmd := suite.helmOutputCommand
			helmOutputCmd.chartName = common.ChartCentralServices
			helmOutputCmd.removeOutputDir = c.removeOutDir
			helmOutputCmd.outputDir = c.outputDir
			if c.createOutDir {
				helmOutputCmd.outputDir = suite.T().TempDir()
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
