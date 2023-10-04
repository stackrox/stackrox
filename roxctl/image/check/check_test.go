package check

import (
	"bytes"
	"context"
	"net"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/summaries/policy"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var (
	// Policies for testing
	// LOW severity
	lowSevPolicy = &storage.Policy{
		Id:          "policy7",
		Name:        "policy 7",
		Description: "policy 7 for testing",
		Remediation: "policy 7 for testing",
		Severity:    storage.Severity_LOW_SEVERITY,
	}
	// MEDIUM severity
	mediumSevPolicy = &storage.Policy{
		Id:          "policy2",
		Name:        "policy 2",
		Description: "policy 2 for testing",
		Remediation: "policy 2 for testing",
		Severity:    storage.Severity_MEDIUM_SEVERITY,
	}
	mediumSevPolicy2 = &storage.Policy{
		Id:          "policy5",
		Name:        "policy 5",
		Description: "policy 5 for testing",
		Remediation: "policy 5 for testing",
		Severity:    storage.Severity_MEDIUM_SEVERITY,
	}
	mediumSevPolicy3 = &storage.Policy{
		Id:          "policy6",
		Name:        "policy 6",
		Description: "policy 6 for testing",
		Remediation: "policy 6 for testing",
		Severity:    storage.Severity_MEDIUM_SEVERITY,
	}
	// HIGH severity
	highSevPolicyWithDeployCreateFail = &storage.Policy{
		Id:          "policy4",
		Name:        "policy 4",
		Description: "policy 4 for testing",
		Remediation: "policy 4 for testing",
		Severity:    storage.Severity_HIGH_SEVERITY,
		EnforcementActions: []storage.EnforcementAction{
			storage.EnforcementAction_FAIL_DEPLOYMENT_CREATE_ENFORCEMENT,
		},
	}
	highSevPolicyWithNoDescription = &storage.Policy{
		Id:          "policy8",
		Name:        "policy 8",
		Remediation: "policy 8 for testing",
		Rationale:   "policy 8 for testing",
		Severity:    storage.Severity_HIGH_SEVERITY,
	}
	// CRITICAL severity
	criticalSevPolicyWithBuildFail = &storage.Policy{
		Id:          "policy1",
		Name:        "policy 1",
		Description: "policy 1 for testing",
		Remediation: "policy 1 for testing",
		Severity:    storage.Severity_CRITICAL_SEVERITY,
		EnforcementActions: []storage.EnforcementAction{
			storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
		},
	}

	// Violation messages for test alerts
	multipleViolationMessages = []*storage.Alert_Violation{
		{
			Message: "test violation 1",
		},
		{
			Message: "test violation 2",
		},
		{
			Message: "test violation 3",
		},
	}
	singleViolationMessage = []*storage.Alert_Violation{
		{
			Message: "test violation 1",
		},
	}

	// Alerts for testing
	testAlertsWithoutFailure = []*storage.Alert{
		{
			Policy:     lowSevPolicy,
			Violations: singleViolationMessage,
		},
		{
			Policy:     mediumSevPolicy,
			Violations: multipleViolationMessages,
		},
		// Alerts with the same policy should result in single policy and merged violation messages
		{
			Policy:     mediumSevPolicy2,
			Violations: multipleViolationMessages,
		},
		{
			Policy:     mediumSevPolicy2,
			Violations: singleViolationMessage,
		},
		{
			Policy:     mediumSevPolicy3,
			Violations: singleViolationMessage,
		},
		// Policy with non build fail Enforcement Action should not result in an error
		{
			Policy:     highSevPolicyWithDeployCreateFail,
			Violations: singleViolationMessage,
		},
		{
			Policy:     highSevPolicyWithNoDescription,
			Violations: multipleViolationMessages,
		},
	}
	testAlertsWithFailure = []*storage.Alert{
		{
			Policy:     lowSevPolicy,
			Violations: singleViolationMessage,
		},
		{
			Policy:     mediumSevPolicy,
			Violations: singleViolationMessage,
		},
		// Alerts with the same policy should result in single policy and merged violation messages
		{
			Policy:     mediumSevPolicy2,
			Violations: multipleViolationMessages,
		},
		{
			Policy:     mediumSevPolicy2,
			Violations: singleViolationMessage,
		},
		{
			Policy:     mediumSevPolicy3,
			Violations: multipleViolationMessages,
		},
		// Policy with non build fail Enforcement Action should not result in an error
		{
			Policy:     highSevPolicyWithDeployCreateFail,
			Violations: singleViolationMessage,
		},
		// Policy with build fail Enforcement Action should result in an error
		{
			Policy:     criticalSevPolicyWithBuildFail,
			Violations: multipleViolationMessages,
		},
		{
			Policy:     highSevPolicyWithNoDescription,
			Violations: multipleViolationMessages,
		},
	}
)

// mock implementation for v1.DetectionServiceServer
type mockDetectionServiceServer struct {
	v1.UnimplementedDetectionServiceServer

	alerts []*storage.Alert
}

func (m *mockDetectionServiceServer) DetectBuildTime(context.Context, *v1.BuildDetectionRequest) (*v1.BuildDetectionResponse, error) {
	return &v1.BuildDetectionResponse{
		Alerts: m.alerts,
	}, nil
}

func TestImageCheckCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(imageCheckTestSuite))
}

type imageCheckTestSuite struct {
	suite.Suite
	imageCheckCommand imageCheckCommand
}

func (suite *imageCheckTestSuite) createGRPCServerWithDetectionService(alerts []*storage.Alert) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	s := grpc.NewServer()
	v1.RegisterDetectionServiceServer(s, &mockDetectionServiceServer{alerts: alerts})
	go func() {
		utils.IgnoreError(func() error { return s.Serve(listener) })
	}()

	conn, _ := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))

	closeF := func() {
		utils.IgnoreError(listener.Close)
		s.Stop()
	}
	return conn, closeF
}

func (suite *imageCheckTestSuite) newTestMockEnvironment(conn *grpc.ClientConn) (environment.Environment, *bytes.Buffer, *bytes.Buffer) {
	return mocks.NewEnvWithConn(conn, suite.T())
}

func (suite *imageCheckTestSuite) SetupTest() {
	suite.imageCheckCommand = imageCheckCommand{
		image:        "nginx:test",
		retryDelay:   3,
		retryCount:   3,
		timeout:      1 * time.Minute,
		retryTimeout: 1 * time.Minute,
	}
}

type outputFormatTest struct {
	shouldFail                 bool
	alerts                     []*storage.Alert
	expectedOutput             string
	expectedErrOutput          string
	expectedErrOutputColorized string
	error                      error
}

func (suite *imageCheckTestSuite) TestCheckImage_TableOutput() {
	cases := map[string]outputFormatTest{
		"should not fail with non build failing enforcement actions": {
			alerts:                     testAlertsWithoutFailure,
			expectedOutput:             "testAlertsWithoutFailure.txt",
			expectedErrOutput:          "WARN:\tA total of 6 policies have been violated\n",
			expectedErrOutputColorized: "\x1b[95mWARN:\tA total of 6 policies have been violated\n\x1b[0m",
		},
		"should fail with build failing enforcement actions": {
			alerts:         testAlertsWithFailure,
			expectedOutput: "testAlertsWithFailure.txt",
			expectedErrOutput: "WARN:\tA total of 7 policies have been violated\n" +
				"ERROR:\tfailed policies found: 1 policies violated that are failing the check\n" +
				"ERROR:\tPolicy \"policy 1\" - Possible remediation: \"policy 1 for testing\"\n",
			expectedErrOutputColorized: "\x1b[95mWARN:\tA total of 7 policies have been violated\n" +
				"\x1b[0m\x1b[31;1mERROR:\tfailed policies found: 1 policies violated that are failing the check\n" +
				"\x1b[0m\x1b[31;1mERROR:\tPolicy \"policy 1\" - Possible remediation: \"policy 1 for testing\"\n\x1b[0m",
			shouldFail: true,
			error:      policy.ErrBreakingPolicies,
		},
	}
	// setup table printer with default options
	tablePrinter, err := printer.NewTabularPrinterFactory(defaultImageCheckHeaders,
		defaultImageCheckJSONPathExpression).CreatePrinter("table")
	suite.Require().NoError(err)
	suite.runOutputTests(cases, tablePrinter, false)
}

func (suite *imageCheckTestSuite) TestCheckImage_JSONOutput() {
	cases := map[string]outputFormatTest{
		"should not fail with non build failing enforcement actions": {
			alerts:         testAlertsWithoutFailure,
			expectedOutput: "testAlertsWithoutFailure.json",
		},
		"should fail with build failing enforcement actions": {
			shouldFail:     true,
			alerts:         testAlertsWithFailure,
			error:          policy.ErrBreakingPolicies,
			expectedOutput: "testAlertsWithFailure.json",
		},
	}

	// setup JSON printer with default options
	jsonPrinter, err := printer.NewJSONPrinterFactory(false, false).CreatePrinter("json")
	suite.Require().NoError(err)
	suite.runOutputTests(cases, jsonPrinter, true)
}

func (suite *imageCheckTestSuite) TestCheckImage_CSVOutput() {
	cases := map[string]outputFormatTest{
		"should not fail with non build failing enforcement actions": {
			alerts:         testAlertsWithoutFailure,
			expectedOutput: "testAlertsWithoutFailure.csv",
		},
		"should fail with build failing enforcement actions": {
			alerts:         testAlertsWithFailure,
			shouldFail:     true,
			error:          policy.ErrBreakingPolicies,
			expectedOutput: "testAlertsWithFailure.csv",
		},
	}

	// setup CSV printer with default options
	csvPrinter, err := printer.NewTabularPrinterFactory(defaultImageCheckHeaders,
		defaultImageCheckJSONPathExpression).CreatePrinter("csv")
	suite.Require().NoError(err)
	suite.runOutputTests(cases, csvPrinter, true)
}

func (suite *imageCheckTestSuite) TestCheckImage_JUnitOutput() {
	cases := map[string]outputFormatTest{
		"should not fail with non build failing enforcement actions": {
			alerts:         testAlertsWithoutFailure,
			expectedOutput: "testAlertsWithoutFailure.xml",
		},
		"should fail with build failing enforcement actions": {
			alerts:         testAlertsWithFailure,
			shouldFail:     true,
			error:          policy.ErrBreakingPolicies,
			expectedOutput: "testAlertsWithFailure.xml",
		},
	}

	// setup CSV printer with default options
	junitPrinter, err := printer.NewJUnitPrinterFactory("image-check", defaultJunitJSONPathExpressions).CreatePrinter("junit")
	suite.Require().NoError(err)
	suite.runOutputTests(cases, junitPrinter, true)
}

func (suite *imageCheckTestSuite) TestConstruct() {
	objectPrinterFactory, err := printer.NewObjectPrinterFactory("json", printer.NewJSONPrinterFactory(false, false))
	suite.Require().NoError(err)
	jsonPrinter, err := printer.NewJSONPrinterFactory(false, false).CreatePrinter("json")
	suite.Require().NoError(err)
	invalidObjectPrinterFactory, err := printer.NewObjectPrinterFactory("json", printer.NewJSONPrinterFactory(false, false))
	suite.Require().NoError(err)
	invalidObjectPrinterFactory.OutputFormat = "table"

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Duration("timeout", 1*time.Minute, "")
	cmd.Flags().Duration("retry-timeout", 1*time.Minute, "")

	cases := map[string]struct {
		shouldFail         bool
		json               bool
		printerFactory     *printer.ObjectPrinterFactory
		printer            printer.ObjectPrinter
		standardizedFormat bool
	}{
		"should return no error if JSON is set and printer should be nil": {
			json: true,
		},
		"should not return an error with a valid ObjectPrinter factory given": {
			printerFactory:     objectPrinterFactory,
			printer:            jsonPrinter,
			standardizedFormat: true,
		},
		"should return an error when invalid values are given for the ObjectPrinter factory": {
			printerFactory: invalidObjectPrinterFactory,
			shouldFail:     true,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			imgCheckCmd := suite.imageCheckCommand
			imgCheckCmd.json = c.json
			err := imgCheckCmd.Construct(nil, cmd, c.printerFactory)
			if c.shouldFail {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
			suite.Assert().Equal(c.printer, imgCheckCmd.objectPrinter)
			suite.Assert().Equal(c.standardizedFormat, imgCheckCmd.standardizedOutputFormat)
			suite.Assert().Equal(1*time.Minute, imgCheckCmd.timeout)
			suite.Assert().Equal(1*time.Minute, imgCheckCmd.retryTimeout)
		})
	}
}

func (suite *imageCheckTestSuite) TestValidate() {
	jsonPrinter, _ := printer.NewJSONPrinterFactory(false, false).CreatePrinter("json")
	cases := map[string]struct {
		printer         printer.ObjectPrinter
		failViolations  bool
		json            bool
		expectedWarning string
	}{
		"failViolations false and json as well, no output expected": {
			expectedWarning: "",
		},
		"failViolations true and json not, warning output expected": {
			failViolations:  true,
			expectedWarning: "WARN:\t--json-fail-on-policy-violations has no effect when --json is not specified.\n",
		},
		"failViolations true and json as well, no output expected": {
			failViolations:  true,
			json:            true,
			expectedWarning: "",
		},
		"failViolations and json are false, jsonPrinter is created": {
			printer:         jsonPrinter,
			expectedWarning: "",
		},
	}
	for name, c := range cases {
		suite.Run(name, func() {
			imgCheckCmd := suite.imageCheckCommand
			imgCheckCmd.json = c.json
			imgCheckCmd.failViolationsWithJSON = c.failViolations
			imgCheckCmd.objectPrinter = c.printer
			testIO, _, _, errOut := io.TestIO()
			imgCheckCmd.env = environment.NewTestCLIEnvironment(suite.T(), testIO, printer.DefaultColorPrinter())
			suite.Assert().NoError(imgCheckCmd.Validate())
			suite.Assert().Equal(c.expectedWarning, errOut.String())
		})
	}
}

func (suite *imageCheckTestSuite) TestLegacyPrint_Error() {
	imgCheckCmd := suite.imageCheckCommand
	env := environment.NewTestCLIEnvironment(suite.T(), io.DiscardIO(), printer.DefaultColorPrinter())
	imgCheckCmd.env = env
	jsonPrinter, _ := printer.NewJSONPrinterFactory(false, false).CreatePrinter("json")

	cases := map[string]struct {
		alerts         []*storage.Alert
		wantErr        bool
		failViolations bool
		json           bool
		printer        printer.ObjectPrinter
	}{
		"non JSON output format no fatal alerts": {
			alerts:  testAlertsWithoutFailure,
			printer: jsonPrinter,
		},
		"non JSON output format fatal alerts": {
			alerts:  testAlertsWithFailure,
			printer: jsonPrinter,
			wantErr: true,
		},
		"legacy JSON output format no fatal alerts": {
			alerts: testAlertsWithoutFailure,
			json:   true,
		},
		"legacy JSON output format fatal alerts": {
			alerts: testAlertsWithFailure,
			json:   true,
		},
		"legacy JSON output format with legacy fail violation flag no fatal alerts": {
			alerts:         testAlertsWithoutFailure,
			json:           true,
			failViolations: true,
		},
		"legacy JSON output format with legacy fail violation flag fatal alerts": {
			alerts:         testAlertsWithFailure,
			json:           true,
			failViolations: true,
			wantErr:        true,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			imgCheckCmd.json = c.json
			imgCheckCmd.failViolationsWithJSON = c.failViolations
			imgCheckCmd.objectPrinter = c.printer
			if err := imgCheckCmd.printResults(c.alerts); (err != nil) != c.wantErr {
				suite.T().Errorf("printResults() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

func (suite *imageCheckTestSuite) TestLegacyPrint_Format() {
	imgCheckCmd := suite.imageCheckCommand

	cases := map[string]struct {
		alerts             []*storage.Alert
		expectedOutput     string
		json               bool
		printAllViolations bool
	}{
		"alert with json output format not failing the build": {
			alerts:             testAlertsWithoutFailure,
			json:               true,
			printAllViolations: true,
			expectedOutput:     "legacy_testAlertsWithoutFailure.json",
		},
		"alert with json output format failing the build": {
			alerts:             testAlertsWithFailure,
			printAllViolations: true,
			json:               true,
			expectedOutput:     "legacy_testAlertsWithFailure.json",
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			testIO, _, out, _ := io.TestIO()
			imgCheckCmd.env = environment.NewTestCLIEnvironment(suite.T(), testIO, printer.DefaultColorPrinter())
			imgCheckCmd.json = c.json
			imgCheckCmd.printAllViolations = c.printAllViolations
			// Errors will be tested within TestLegacyPrint_Error
			_ = imgCheckCmd.printResults(c.alerts)
			expectedOutput, err := os.ReadFile(path.Join("testdata", c.expectedOutput))
			suite.Require().NoError(err)
			suite.Assert().Equal(string(expectedOutput), out.String())
		})
	}
}

// helper to run output format tests
func (suite *imageCheckTestSuite) runOutputTests(cases map[string]outputFormatTest, printer printer.ObjectPrinter,
	standardizedFormat bool,
) {
	const colorTestPrefix = "color_"
	for name, c := range cases {
		suite.Run(name, func() {
			out, errOut, closeF, imgCheckCmd := suite.createNewImgCheckCmd(c, printer, standardizedFormat)
			defer closeF()
			suite.assertError(imgCheckCmd, c)
			expectedOutput, err := os.ReadFile(path.Join("testdata", c.expectedOutput))
			suite.Require().NoError(err)
			suite.Assert().Equal(string(expectedOutput), out.String())
			suite.Assert().Equal(c.expectedErrOutput, errOut.String())
		})
		suite.Run(colorTestPrefix+name, func() {
			if runtime.GOOS == "windows" {
				suite.T().Skip("Windows has different color sequences than Linux/Mac.")
			}
			color.NoColor = false
			defer func() { color.NoColor = true }()

			out, errOut, closeF, imgCheckCmd := suite.createNewImgCheckCmd(c, printer, standardizedFormat)
			defer closeF()
			suite.assertError(imgCheckCmd, c)
			expectedOutput, err := os.ReadFile(path.Join("testdata", colorTestPrefix+c.expectedOutput))
			suite.Require().NoError(err)
			suite.Assert().Equal(string(expectedOutput), out.String())
			suite.Assert().Equal(c.expectedErrOutputColorized, errOut.String())
		})
	}
}

func (suite *imageCheckTestSuite) assertError(imgCheckCmd imageCheckCommand, c outputFormatTest) {
	err := imgCheckCmd.CheckImage()
	if c.shouldFail {
		suite.Require().Error(err)
		suite.Assert().ErrorIs(err, c.error)
	} else {
		suite.Require().NoError(err)
	}
}

func (suite *imageCheckTestSuite) createNewImgCheckCmd(c outputFormatTest, printer printer.ObjectPrinter, standardizedFormat bool) (*bytes.Buffer, *bytes.Buffer, func(), imageCheckCommand) {
	var out *bytes.Buffer
	var errOut *bytes.Buffer
	conn, closeF := suite.createGRPCServerWithDetectionService(c.alerts)

	imgCheckCmd := suite.imageCheckCommand
	imgCheckCmd.objectPrinter = printer
	imgCheckCmd.standardizedOutputFormat = standardizedFormat

	imgCheckCmd.env, out, errOut = suite.newTestMockEnvironment(conn)
	return out, errOut, closeF, imgCheckCmd
}
