package check

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/summaries/policy"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
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
	alerts []*storage.Alert
	// This will allow us to use the struct when registering it via v1.RegisterDetectionServiceServer without the need
	// to implement all functions of the interface, only the one we require for testing.
	v1.DetectionServiceServer
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
	}), grpc.WithInsecure())

	closeF := func() {
		utils.IgnoreError(listener.Close)
		s.Stop()
	}
	return conn, closeF
}

func (suite *imageCheckTestSuite) newTestMockEnvironment(conn *grpc.ClientConn) (environment.Environment, *bytes.Buffer) {
	envMock := mocks.NewMockEnvironment(gomock.NewController(suite.T()))

	testIO, _, out, _ := environment.TestIO()
	logger := environment.NewLogger(testIO, printer.DefaultColorPrinter())

	envMock.EXPECT().Logger().AnyTimes().Return(logger)
	envMock.EXPECT().InputOutput().AnyTimes().Return(testIO)
	envMock.EXPECT().GRPCConnection().AnyTimes().Return(conn, nil)
	return envMock, out
}

func (suite *imageCheckTestSuite) SetupTest() {
	suite.imageCheckCommand = imageCheckCommand{
		image:      "nginx:test",
		retryDelay: 3,
		timeout:    1 * time.Minute,
	}
}

type outputFormatTest struct {
	shouldFail        bool
	alerts            []*storage.Alert
	expectedOutput    string
	expectedErrOutput string
	error             error
}

func (suite *imageCheckTestSuite) TestCheckImage_TableOutput() {
	cases := map[string]outputFormatTest{
		"should not fail with non build failing enforcement actions": {
			alerts: testAlertsWithoutFailure,
			expectedOutput: `Policy check results for image: nginx:test
(TOTAL: 6, LOW: 1, MEDIUM: 3, HIGH: 2, CRITICAL: 0)

+----------+----------+--------------+----------------------+--------------------+----------------------+
|  POLICY  | SEVERITY | BREAKS BUILD |     DESCRIPTION      |     VIOLATION      |     REMEDIATION      |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 4 |   HIGH   |      -       | policy 4 for testing | - test violation 1 | policy 4 for testing |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 8 |   HIGH   |      -       |          -           | - test violation 1 | policy 8 for testing |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 2 |                      |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 3 |                      |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 2 |  MEDIUM  |      -       | policy 2 for testing | - test violation 1 | policy 2 for testing |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 2 |                      |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 3 |                      |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 5 |  MEDIUM  |      -       | policy 5 for testing | - test violation 1 | policy 5 for testing |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 2 |                      |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 3 |                      |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 1 |                      |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 6 |  MEDIUM  |      -       | policy 6 for testing | - test violation 1 | policy 6 for testing |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 7 |   LOW    |      -       | policy 7 for testing | - test violation 1 | policy 7 for testing |
+----------+----------+--------------+----------------------+--------------------+----------------------+
`,
			expectedErrOutput: "WARN: A total of 6 policies have been violated\n",
		},
		"should fail with build failing enforcement actions": {
			alerts: testAlertsWithFailure,
			expectedOutput: `Policy check results for image: nginx:test
(TOTAL: 7, LOW: 1, MEDIUM: 3, HIGH: 2, CRITICAL: 1)

+----------+----------+--------------+----------------------+--------------------+----------------------+
|  POLICY  | SEVERITY | BREAKS BUILD |     DESCRIPTION      |     VIOLATION      |     REMEDIATION      |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 1 | CRITICAL |      X       | policy 1 for testing | - test violation 1 | policy 1 for testing |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 2 |                      |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 3 |                      |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 4 |   HIGH   |      -       | policy 4 for testing | - test violation 1 | policy 4 for testing |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 8 |   HIGH   |      -       |          -           | - test violation 1 | policy 8 for testing |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 2 |                      |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 3 |                      |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 2 |  MEDIUM  |      -       | policy 2 for testing | - test violation 1 | policy 2 for testing |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 5 |  MEDIUM  |      -       | policy 5 for testing | - test violation 1 | policy 5 for testing |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 2 |                      |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 3 |                      |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 1 |                      |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 6 |  MEDIUM  |      -       | policy 6 for testing | - test violation 1 | policy 6 for testing |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 2 |                      |
|          |          |              |                      |                    |                      |
|          |          |              |                      | - test violation 3 |                      |
+----------+----------+--------------+----------------------+--------------------+----------------------+
| policy 7 |   LOW    |      -       | policy 7 for testing | - test violation 1 | policy 7 for testing |
+----------+----------+--------------+----------------------+--------------------+----------------------+
`,
			expectedErrOutput: `WARN: A total of 7 policies have been violated
ERROR: failed policies found: 1 policies violated that are failing the check
ERROR: Policy "policy 1" - Possible remediation: "policy 1 for testing"
`,
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
			alerts: testAlertsWithoutFailure,
			expectedOutput: `{
  "results": [
    {
      "metadata": {
        "id": "unknown",
        "additionalInfo": null
      },
      "summary": {
        "CRITICAL": 0,
        "HIGH": 2,
        "LOW": 1,
        "MEDIUM": 3,
        "TOTAL": 6
      },
      "violatedPolicies": [
        {
          "name": "policy 4",
          "severity": "HIGH",
          "description": "policy 4 for testing",
          "violation": [
            "test violation 1"
          ],
          "remediation": "policy 4 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 8",
          "severity": "HIGH",
          "description": "",
          "violation": [
            "test violation 1",
            "test violation 2",
            "test violation 3"
          ],
          "remediation": "policy 8 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 2",
          "severity": "MEDIUM",
          "description": "policy 2 for testing",
          "violation": [
            "test violation 1",
            "test violation 2",
            "test violation 3"
          ],
          "remediation": "policy 2 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 5",
          "severity": "MEDIUM",
          "description": "policy 5 for testing",
          "violation": [
            "test violation 1",
            "test violation 2",
            "test violation 3",
            "test violation 1"
          ],
          "remediation": "policy 5 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 6",
          "severity": "MEDIUM",
          "description": "policy 6 for testing",
          "violation": [
            "test violation 1"
          ],
          "remediation": "policy 6 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 7",
          "severity": "LOW",
          "description": "policy 7 for testing",
          "violation": [
            "test violation 1"
          ],
          "remediation": "policy 7 for testing",
          "failingCheck": false
        }
      ]
    }
  ],
  "summary": {
    "CRITICAL": 0,
    "HIGH": 2,
    "LOW": 1,
    "MEDIUM": 3,
    "TOTAL": 6
  }
}
`,
		},
		"should fail with build failing enforcement actions": {
			shouldFail: true,
			alerts:     testAlertsWithFailure,
			error:      policy.ErrBreakingPolicies,
			expectedOutput: `{
  "results": [
    {
      "metadata": {
        "id": "unknown",
        "additionalInfo": null
      },
      "summary": {
        "CRITICAL": 1,
        "HIGH": 2,
        "LOW": 1,
        "MEDIUM": 3,
        "TOTAL": 7
      },
      "violatedPolicies": [
        {
          "name": "policy 1",
          "severity": "CRITICAL",
          "description": "policy 1 for testing",
          "violation": [
            "test violation 1",
            "test violation 2",
            "test violation 3"
          ],
          "remediation": "policy 1 for testing",
          "failingCheck": true
        },
        {
          "name": "policy 4",
          "severity": "HIGH",
          "description": "policy 4 for testing",
          "violation": [
            "test violation 1"
          ],
          "remediation": "policy 4 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 8",
          "severity": "HIGH",
          "description": "",
          "violation": [
            "test violation 1",
            "test violation 2",
            "test violation 3"
          ],
          "remediation": "policy 8 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 2",
          "severity": "MEDIUM",
          "description": "policy 2 for testing",
          "violation": [
            "test violation 1"
          ],
          "remediation": "policy 2 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 5",
          "severity": "MEDIUM",
          "description": "policy 5 for testing",
          "violation": [
            "test violation 1",
            "test violation 2",
            "test violation 3",
            "test violation 1"
          ],
          "remediation": "policy 5 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 6",
          "severity": "MEDIUM",
          "description": "policy 6 for testing",
          "violation": [
            "test violation 1",
            "test violation 2",
            "test violation 3"
          ],
          "remediation": "policy 6 for testing",
          "failingCheck": false
        },
        {
          "name": "policy 7",
          "severity": "LOW",
          "description": "policy 7 for testing",
          "violation": [
            "test violation 1"
          ],
          "remediation": "policy 7 for testing",
          "failingCheck": false
        }
      ]
    }
  ],
  "summary": {
    "CRITICAL": 1,
    "HIGH": 2,
    "LOW": 1,
    "MEDIUM": 3,
    "TOTAL": 7
  }
}
`,
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
			alerts: testAlertsWithoutFailure,
			expectedOutput: `POLICY,SEVERITY,BREAKS BUILD,DESCRIPTION,VIOLATION,REMEDIATION
policy 4,HIGH,-,policy 4 for testing,- test violation 1,policy 4 for testing
policy 8,HIGH,-,-,"- test violation 1
- test violation 2
- test violation 3",policy 8 for testing
policy 2,MEDIUM,-,policy 2 for testing,"- test violation 1
- test violation 2
- test violation 3",policy 2 for testing
policy 5,MEDIUM,-,policy 5 for testing,"- test violation 1
- test violation 2
- test violation 3
- test violation 1",policy 5 for testing
policy 6,MEDIUM,-,policy 6 for testing,- test violation 1,policy 6 for testing
policy 7,LOW,-,policy 7 for testing,- test violation 1,policy 7 for testing
`,
		},
		"should fail with build failing enforcement actions": {
			alerts:     testAlertsWithFailure,
			shouldFail: true,
			error:      policy.ErrBreakingPolicies,
			expectedOutput: `POLICY,SEVERITY,BREAKS BUILD,DESCRIPTION,VIOLATION,REMEDIATION
policy 1,CRITICAL,X,policy 1 for testing,"- test violation 1
- test violation 2
- test violation 3",policy 1 for testing
policy 4,HIGH,-,policy 4 for testing,- test violation 1,policy 4 for testing
policy 8,HIGH,-,-,"- test violation 1
- test violation 2
- test violation 3",policy 8 for testing
policy 2,MEDIUM,-,policy 2 for testing,- test violation 1,policy 2 for testing
policy 5,MEDIUM,-,policy 5 for testing,"- test violation 1
- test violation 2
- test violation 3
- test violation 1",policy 5 for testing
policy 6,MEDIUM,-,policy 6 for testing,"- test violation 1
- test violation 2
- test violation 3",policy 6 for testing
policy 7,LOW,-,policy 7 for testing,- test violation 1,policy 7 for testing
`,
		},
	}

	// setup CSV printer with default options
	csvPrinter, err := printer.NewTabularPrinterFactory(defaultImageCheckHeaders,
		defaultImageCheckJSONPathExpression).CreatePrinter("csv")
	suite.Require().NoError(err)
	suite.runOutputTests(cases, csvPrinter, true)
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
			testIO, _, _, errOut := environment.TestIO()
			imgCheckCmd.env = environment.NewCLIEnvironment(testIO, printer.DefaultColorPrinter())
			suite.Assert().NoError(imgCheckCmd.Validate())
			suite.Assert().Equal(c.expectedWarning, errOut.String())
		})
	}
}

func (suite *imageCheckTestSuite) TestLegacyPrint_Error() {
	imgCheckCmd := suite.imageCheckCommand
	env := environment.NewCLIEnvironment(environment.DiscardIO(), printer.DefaultColorPrinter())
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
			expectedOutput: `{
  "alerts": [
    {
      "policy": {
        "id": "policy7",
        "name": "policy 7",
        "description": "policy 7 for testing",
        "remediation": "policy 7 for testing",
        "severity": "LOW_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        }
      ]
    },
    {
      "policy": {
        "id": "policy2",
        "name": "policy 2",
        "description": "policy 2 for testing",
        "remediation": "policy 2 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        },
        {
          "message": "test violation 2"
        },
        {
          "message": "test violation 3"
        }
      ]
    },
    {
      "policy": {
        "id": "policy5",
        "name": "policy 5",
        "description": "policy 5 for testing",
        "remediation": "policy 5 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        },
        {
          "message": "test violation 2"
        },
        {
          "message": "test violation 3"
        }
      ]
    },
    {
      "policy": {
        "id": "policy5",
        "name": "policy 5",
        "description": "policy 5 for testing",
        "remediation": "policy 5 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        }
      ]
    },
    {
      "policy": {
        "id": "policy6",
        "name": "policy 6",
        "description": "policy 6 for testing",
        "remediation": "policy 6 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        }
      ]
    },
    {
      "policy": {
        "id": "policy4",
        "name": "policy 4",
        "description": "policy 4 for testing",
        "remediation": "policy 4 for testing",
        "severity": "HIGH_SEVERITY",
        "enforcementActions": [
          "FAIL_DEPLOYMENT_CREATE_ENFORCEMENT"
        ]
      },
      "violations": [
        {
          "message": "test violation 1"
        }
      ]
    },
    {
      "policy": {
        "id": "policy8",
        "name": "policy 8",
        "rationale": "policy 8 for testing",
        "remediation": "policy 8 for testing",
        "severity": "HIGH_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        },
        {
          "message": "test violation 2"
        },
        {
          "message": "test violation 3"
        }
      ]
    }
  ]
}
`,
		},
		"alert with json output format failing the build": {
			alerts:             testAlertsWithFailure,
			printAllViolations: true,
			json:               true,
			expectedOutput: `{
  "alerts": [
    {
      "policy": {
        "id": "policy7",
        "name": "policy 7",
        "description": "policy 7 for testing",
        "remediation": "policy 7 for testing",
        "severity": "LOW_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        }
      ]
    },
    {
      "policy": {
        "id": "policy2",
        "name": "policy 2",
        "description": "policy 2 for testing",
        "remediation": "policy 2 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        }
      ]
    },
    {
      "policy": {
        "id": "policy5",
        "name": "policy 5",
        "description": "policy 5 for testing",
        "remediation": "policy 5 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        },
        {
          "message": "test violation 2"
        },
        {
          "message": "test violation 3"
        }
      ]
    },
    {
      "policy": {
        "id": "policy5",
        "name": "policy 5",
        "description": "policy 5 for testing",
        "remediation": "policy 5 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        }
      ]
    },
    {
      "policy": {
        "id": "policy6",
        "name": "policy 6",
        "description": "policy 6 for testing",
        "remediation": "policy 6 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        },
        {
          "message": "test violation 2"
        },
        {
          "message": "test violation 3"
        }
      ]
    },
    {
      "policy": {
        "id": "policy4",
        "name": "policy 4",
        "description": "policy 4 for testing",
        "remediation": "policy 4 for testing",
        "severity": "HIGH_SEVERITY",
        "enforcementActions": [
          "FAIL_DEPLOYMENT_CREATE_ENFORCEMENT"
        ]
      },
      "violations": [
        {
          "message": "test violation 1"
        }
      ]
    },
    {
      "policy": {
        "id": "policy1",
        "name": "policy 1",
        "description": "policy 1 for testing",
        "remediation": "policy 1 for testing",
        "severity": "CRITICAL_SEVERITY",
        "enforcementActions": [
          "FAIL_BUILD_ENFORCEMENT"
        ]
      },
      "violations": [
        {
          "message": "test violation 1"
        },
        {
          "message": "test violation 2"
        },
        {
          "message": "test violation 3"
        }
      ]
    },
    {
      "policy": {
        "id": "policy8",
        "name": "policy 8",
        "rationale": "policy 8 for testing",
        "remediation": "policy 8 for testing",
        "severity": "HIGH_SEVERITY"
      },
      "violations": [
        {
          "message": "test violation 1"
        },
        {
          "message": "test violation 2"
        },
        {
          "message": "test violation 3"
        }
      ]
    }
  ]
}
`,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			testIO, _, out, _ := environment.TestIO()
			imgCheckCmd.env = environment.NewCLIEnvironment(testIO, printer.DefaultColorPrinter())
			imgCheckCmd.json = c.json
			imgCheckCmd.printAllViolations = c.printAllViolations
			// Errors will be tested within TestLegacyPrint_Error
			_ = imgCheckCmd.printResults(c.alerts)
			suite.Assert().Equal(c.expectedOutput, out.String())
		})
	}
}

// helper to run output format tests
func (suite *imageCheckTestSuite) runOutputTests(cases map[string]outputFormatTest, printer printer.ObjectPrinter,
	standardizedFormat bool) {
	for name, c := range cases {
		suite.Run(name, func() {
			var out *bytes.Buffer
			conn, closeF := suite.createGRPCServerWithDetectionService(c.alerts)
			defer closeF()

			imgCheckCmd := suite.imageCheckCommand
			imgCheckCmd.objectPrinter = printer
			imgCheckCmd.standardizedOutputFormat = standardizedFormat

			imgCheckCmd.env, out = suite.newTestMockEnvironment(conn)
			err := imgCheckCmd.CheckImage()
			if c.shouldFail {
				suite.Require().Error(err)
				suite.Assert().ErrorIs(err, c.error)
			} else {
				suite.Require().NoError(err)

			}
			suite.Assert().Equal(c.expectedOutput, out.String())
		})
	}
}
