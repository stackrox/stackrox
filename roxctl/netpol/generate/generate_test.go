package generate

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stretchr/testify/suite"
)

func TestGenerateNetpolCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(generateNetpolTestSuite))
}

type generateNetpolTestSuite struct {
	suite.Suite
}

func (d *generateNetpolTestSuite) TestGenerateNetpol() {
	cases := map[string]struct {
		inputFolderPath       string
		stopOnFirstErr        bool
		treatWarningsAsErrors bool
		outFile               string
		outDir                string
		removeOutputPath      bool
		dnsPort               *string

		expectedValidationError error
		expectedWarnings        []string
		expectedErrors          []string
	}{
		"not existing inputFolderPath should raise 'does not exist' error": {
			inputFolderPath: "/tmp/xxx",

			expectedValidationError: nil,
			expectedWarnings:        []string{},
			expectedErrors: []string{
				"the path \"/tmp/xxx\" does not exist",
				"error generating network policies: could not find any Kubernetes workload resources",
			},
		},
		"happyPath": {
			inputFolderPath:         "testdata/minimal",
			expectedValidationError: nil,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"empty yamls should yield error no kubernetes resources found": {
			inputFolderPath:         "testdata/empty-yamls",
			expectedValidationError: nil,
			expectedWarnings: []string{
				"unable to decode \"testdata/empty-yamls/empty.yaml\": Object 'Kind' is missing in",
				"unable to decode \"testdata/empty-yamls/empty2.yaml\": Object 'Kind' is missing in"},
			expectedErrors: []string{"could not find any Kubernetes workload resources"},
		},
		"generation should stop on first warning when warnings are treated as errors ": {
			inputFolderPath:         "testdata/dirty",
			stopOnFirstErr:          true,
			treatWarningsAsErrors:   true,
			expectedValidationError: nil,
			expectedErrors:          []string{"could not find any Kubernetes workload resources"},
			expectedWarnings: []string{
				"error parsing testdata/dirty/backend.yaml",
				"error parsing testdata/dirty/frontend.yaml",
			},
		},
		"output should be written to a single file": {
			inputFolderPath:         "testdata/minimal",
			outFile:                 d.T().TempDir() + "/out.yaml",
			outDir:                  "",
			removeOutputPath:        false,
			expectedValidationError: nil,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"output should be written to files in a directory": {
			inputFolderPath:         "testdata/minimal",
			outFile:                 "",
			outDir:                  d.T().TempDir(),
			removeOutputPath:        true,
			expectedValidationError: nil,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"should return error that the dir already exists": {
			inputFolderPath:  "testdata/minimal",
			outFile:          "",
			outDir:           d.T().TempDir(),
			removeOutputPath: false,

			expectedValidationError: errox.AlreadyExists,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"should report bad port name": {
			inputFolderPath: "testdata/minimal",
			dnsPort:         ptrFromString("bad@chars"),

			expectedValidationError: errox.InvalidArgs,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"should report empty string as bad port name": {
			inputFolderPath: "testdata/minimal",
			dnsPort:         ptrFromString(""),

			expectedValidationError: errox.InvalidArgs,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"should report bad port number": {
			inputFolderPath: "testdata/minimal",
			dnsPort:         ptrFromString("100000"),

			expectedValidationError: errox.InvalidArgs,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"should report 0 as a bad port number": {
			inputFolderPath: "testdata/minimal",
			dnsPort:         ptrFromString("0"),

			expectedValidationError: errox.InvalidArgs,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"should report a negative port as a bad port number": {
			inputFolderPath: "testdata/minimal",
			dnsPort:         ptrFromString("-17"),

			expectedValidationError: errox.InvalidArgs,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
	}

	for name, tt := range cases {
		d.Run(name, func() {
			testCmd := &cobra.Command{Use: "test"}
			testCmd.Flags().String("output-dir", "", "")
			testCmd.Flags().String("output-file", "", "")
			testCmd.Flags().String("dnsport", "", "")

			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			generateNetpolCmd := netpolGenerateCmd{
				Options: NetpolGenerateOptions{
					StopOnFirstError:      tt.stopOnFirstErr,
					TreatWarningsAsErrors: false, // this is tested in netpol/resources/npg_test.go
					OutputFolderPath:      tt.outDir,
					OutputFilePath:        tt.outFile,
					RemoveOutputPath:      tt.removeOutputPath,
				},
				offline:         true,
				inputFolderPath: "", // set through construct
				env:             env,
				printer:         nil,
			}
			if tt.outDir != "" {
				d.Assert().NoError(testCmd.Flags().Set("output-dir", tt.outDir))
			}
			if tt.outFile != "" {
				d.Assert().NoError(testCmd.Flags().Set("output-file", tt.outFile))
			}
			if tt.dnsPort != nil {
				d.Assert().NoError(testCmd.Flags().Set("dnsport", *tt.dnsPort))
				generateNetpolCmd.Options.DNSPort = *tt.dnsPort
			}

			generator, err := generateNetpolCmd.construct([]string{tt.inputFolderPath}, testCmd)
			d.Assert().NoError(err)

			err = generateNetpolCmd.validate()
			if tt.expectedValidationError != nil {
				d.Require().Error(err, "validation error is expected")
				d.Assert().ErrorIs(err, tt.expectedValidationError)
				return
			}
			d.Assert().NoError(err)

			warns, errs := generateNetpolCmd.generateNetpol(generator)
			npg.AssertErrorsContain(d.T(), tt.expectedErrors, errs, "errors")
			npg.AssertErrorsContain(d.T(), tt.expectedWarnings, warns, "warnings")
		})
	}
}

func ptrFromString(s string) *string {
	return &s
}
