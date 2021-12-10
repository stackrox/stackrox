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
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
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
		Rationale:   "policy 7 for testing",
		Severity:    storage.Severity_LOW_SEVERITY,
	}

	// MEDIUM severity
	mediumSevPolicy = &storage.Policy{
		Id:          "policy2",
		Name:        "policy 2",
		Description: "policy 2 for testing",
		Remediation: "policy 2 for testing",
		Rationale:   "policy 2 for testing",
		Severity:    storage.Severity_MEDIUM_SEVERITY,
	}
	mediumSevPolicy2 = &storage.Policy{
		Id:          "policy5",
		Name:        "policy 5",
		Description: "policy 5 for testing",
		Remediation: "policy 5 for testing",
		Rationale:   "policy 5 for testing",
		Severity:    storage.Severity_MEDIUM_SEVERITY,
	}
	mediumSevPolicy3 = &storage.Policy{
		Id:          "policy6",
		Name:        "policy 6",
		Description: "policy 6 for testing",
		Remediation: "policy 6 for testing",
		Rationale:   "policy 6 for testing",
		Severity:    storage.Severity_MEDIUM_SEVERITY,
	}

	// HIGH severity
	highSevPolicyWithDeployScaleZero = &storage.Policy{
		Id:          "policy4",
		Name:        "policy 4",
		Description: "policy 4 for testing",
		Remediation: "policy 4 for testing",
		Rationale:   "policy 4 for testing",
		Severity:    storage.Severity_HIGH_SEVERITY,
		EnforcementActions: []storage.EnforcementAction{
			storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
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
		Rationale:   "policy 1 for testing",
		Severity:    storage.Severity_CRITICAL_SEVERITY,
		EnforcementActions: []storage.EnforcementAction{
			storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
		},
	}

	singleViolationMessage = []*storage.Alert_Violation{
		{
			Message: "testing alert violation message",
		},
	}
	multipleViolationMessages = []*storage.Alert_Violation{
		{
			Message: "testing multiple alert violation messages 1",
		},
		{
			Message: "testing multiple alert violation messages 2",
		},
		{
			Message: "testing multiple alert violation messages 3",
		},
	}

	testDeploymentEntity = &storage.Alert_Deployment_{
		Deployment: &storage.Alert_Deployment{
			Name: "wordpress",
			Type: "Deployment",
		},
	}

	testDeploymentAlertsWithFailure = []*storage.Alert{
		{
			Entity:     testDeploymentEntity,
			Policy:     lowSevPolicy,
			Violations: singleViolationMessage,
		},
		{
			Policy:     mediumSevPolicy,
			Entity:     testDeploymentEntity,
			Violations: multipleViolationMessages,
		},
		// multiple alerts with same policies should result in single policy violation
		// and their violation messages should be merged
		{
			Policy:     mediumSevPolicy2,
			Entity:     testDeploymentEntity,
			Violations: singleViolationMessage,
		},
		{
			Policy:     mediumSevPolicy2,
			Entity:     testDeploymentEntity,
			Violations: multipleViolationMessages,
		},
		{
			Policy:     mediumSevPolicy3,
			Entity:     testDeploymentEntity,
			Violations: singleViolationMessage,
		},
		{
			Policy:     highSevPolicyWithDeployScaleZero,
			Entity:     testDeploymentEntity,
			Violations: multipleViolationMessages,
		},
		{
			Policy:     highSevPolicyWithNoDescription,
			Entity:     testDeploymentEntity,
			Violations: multipleViolationMessages,
		},
	}

	testDeploymentAlertsWithoutFailure = []*storage.Alert{
		{
			Entity:     testDeploymentEntity,
			Policy:     lowSevPolicy,
			Violations: singleViolationMessage,
		},
		{
			Policy:     mediumSevPolicy,
			Entity:     testDeploymentEntity,
			Violations: multipleViolationMessages,
		},
		// multiple alerts with same policies should result in single policy violation
		// and their violation messages should be merged
		{
			Policy:     mediumSevPolicy2,
			Entity:     testDeploymentEntity,
			Violations: singleViolationMessage,
		},
		{
			Policy:     mediumSevPolicy2,
			Entity:     testDeploymentEntity,
			Violations: multipleViolationMessages,
		},
		{
			Policy:     mediumSevPolicy3,
			Entity:     testDeploymentEntity,
			Violations: singleViolationMessage,
		},
		// alert with policy which is NOT storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT should not result in a
		// failure
		{
			Policy:     criticalSevPolicyWithBuildFail,
			Entity:     testDeploymentEntity,
			Violations: multipleViolationMessages,
		},
		{
			Policy:     highSevPolicyWithNoDescription,
			Entity:     testDeploymentEntity,
			Violations: multipleViolationMessages,
		},
	}
)

// mock for testing implementing v1.DetectionServiceServer
type mockDetectionServiceServer struct {
	v1.DetectionServiceServer
	alerts []*storage.Alert
}

func (m *mockDetectionServiceServer) DetectDeployTimeFromYAML(ctx context.Context, req *v1.DeployYAMLDetectionRequest) (*v1.DeployDetectionResponse, error) {
	return &v1.DeployDetectionResponse{
		Runs: []*v1.DeployDetectionResponse_Run{
			{
				Alerts: m.alerts,
			},
		},
	}, nil
}

func TestDeploymentCheckCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(deployCheckTestSuite))
}

type deployCheckTestSuite struct {
	suite.Suite
	defaultDeploymentCheckCommand deploymentCheckCommand
}

func (d *deployCheckTestSuite) createGRPCMockDetectionService(alerts []*storage.Alert) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	v1.RegisterDetectionServiceServer(server, &mockDetectionServiceServer{alerts: alerts})

	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithInsecure())
	d.Require().NoError(err)

	closeFunction := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, closeFunction
}

func (d *deployCheckTestSuite) createMockEnvironmentWithConn(conn *grpc.ClientConn) (environment.Environment, *bytes.Buffer) {
	envMock := mocks.NewMockEnvironment(gomock.NewController(d.T()))

	testIO, _, out, _ := environment.TestIO()
	env := environment.NewCLIEnvironment(testIO, printer.DefaultColorPrinter())

	envMock.EXPECT().InputOutput().AnyTimes().Return(env.InputOutput())
	envMock.EXPECT().Logger().AnyTimes().Return(env.Logger())
	envMock.EXPECT().GRPCConnection().AnyTimes().Return(conn, nil)
	envMock.EXPECT().ColorWriter().AnyTimes().Return(env.ColorWriter())

	return envMock, out
}

func (d *deployCheckTestSuite) SetupTest() {
	d.defaultDeploymentCheckCommand = deploymentCheckCommand{
		file:               "testdata/deployment.yaml",
		retryDelay:         3,
		retryCount:         3,
		timeout:            1 * time.Minute,
		printAllViolations: true,
	}
}

func (d *deployCheckTestSuite) TestConstruct() {
	jsonPrinter, err := printer.NewJSONPrinterFactory(false, false).CreatePrinter("json")
	d.Require().NoError(err)
	validObjectPrinterFactory, err := printer.NewObjectPrinterFactory("json",
		printer.NewJSONPrinterFactory(false, false))
	d.Require().NoError(err)
	invalidObjectPrinterFactory, err := printer.NewObjectPrinterFactory("json",
		printer.NewJSONPrinterFactory(false, false))
	d.Require().NoError(err)
	invalidObjectPrinterFactory.OutputFormat = "table"

	expectedTimeout := 10 * time.Minute

	testCmd := &cobra.Command{Use: "test"}
	testCmd.Flags().Duration("timeout", expectedTimeout, "")

	cases := map[string]struct {
		timeout    time.Duration
		f          *printer.ObjectPrinterFactory
		p          printer.ObjectPrinter
		json       bool
		shouldFail bool
		error      error
	}{
		"should not fail and create printer": {
			timeout: expectedTimeout,
			f:       validObjectPrinterFactory,
			p:       jsonPrinter,
		},
		"should not create a printer when using legacy json output": {
			timeout: expectedTimeout,
			f:       validObjectPrinterFactory,
			json:    true,
		},
		"should fail when invalid values are provided for object printer factory": {
			timeout:    expectedTimeout,
			f:          invalidObjectPrinterFactory,
			shouldFail: true,
			error:      errorhelpers.ErrInvalidArgs,
		},
	}

	for name, c := range cases {
		d.Run(name, func() {
			deployCheckCmd := d.defaultDeploymentCheckCommand
			deployCheckCmd.json = c.json

			err := deployCheckCmd.Construct(nil, testCmd, c.f)
			if c.shouldFail {
				d.Require().Error(err)
				d.Assert().ErrorIs(err, c.error)
			} else {
				d.Assert().NoError(err)
			}
			d.Assert().Equal(c.p, deployCheckCmd.printer)
		})
	}
}

func (d *deployCheckTestSuite) TestValidate() {
	cases := map[string]struct {
		file       string
		shouldFail bool
		error      error
	}{
		"should not fail with default file name": {
			file: d.defaultDeploymentCheckCommand.file,
		},
		"should fail with non existing file name": {
			file:       "invalidfile",
			shouldFail: true,
			error:      errorhelpers.ErrInvalidArgs,
		},
	}

	for name, c := range cases {
		d.Run(name, func() {
			deployCheckCmd := d.defaultDeploymentCheckCommand
			deployCheckCmd.file = c.file

			err := deployCheckCmd.Validate()
			if c.shouldFail {
				d.Require().Error(err)
				d.Assert().ErrorIs(err, c.error)
			} else {
				d.Assert().NoError(err)
			}
		})
	}
}

type outputFormatTest struct {
	alerts            []*storage.Alert
	expectedOutput    string
	expectedErrOutput string
	shouldFail        bool
	error             error
}

func (d *deployCheckTestSuite) TestCheck_TableOutput() {
	cases := map[string]outputFormatTest{
		"should not fail with non failing enforcement actions": {
			alerts:            testDeploymentAlertsWithoutFailure,
			expectedOutput:    "testDeploymentAlertsWithoutFailure.txt",
			expectedErrOutput: "WARN: A total of 6 policies have been violated\n",
		},
		"should fail with failing enforcement actions": {
			alerts:         testDeploymentAlertsWithFailure,
			expectedOutput: "testDeploymentAlertsWithFailure.txt",
			expectedErrOutput: `WARN: A total of 6 policies have been violated
ERROR: failed policies found: 1 policies violated that are failing the check
ERROR: Policy "policy 4" within Deployment "wordpress" - Possible remediation: "policy 4"
`,
			error:      policy.ErrBreakingPolicies,
			shouldFail: true,
		},
	}

	tablePrinter, err := printer.NewTabularPrinterFactory(defaultDeploymentCheckHeaders,
		defaultDeploymentCheckJSONPathExpression).CreatePrinter("table")
	d.Require().NoError(err)
	d.runOutputTests(cases, tablePrinter, false)
}

func (d *deployCheckTestSuite) TestCheck_JSONOutput() {
	cases := map[string]outputFormatTest{
		"should not fail with non failing enforcement actions": {
			alerts:         testDeploymentAlertsWithoutFailure,
			expectedOutput: "testDeploymentAlertsWithoutFailure.json",
		},
		"should fail with failing enforcement actions": {
			alerts:         testDeploymentAlertsWithFailure,
			expectedOutput: "testDeploymentAlertsWithFailure.json",
			shouldFail:     true,
			error:          policy.ErrBreakingPolicies,
		},
	}

	jsonPrinter, err := printer.NewJSONPrinterFactory(false, false).CreatePrinter("json")
	d.Require().NoError(err)
	d.runOutputTests(cases, jsonPrinter, true)
}

func (d *deployCheckTestSuite) TestCheck_CSVOutput() {
	cases := map[string]outputFormatTest{
		"should not fail with non failing enforcement actions": {
			alerts:         testDeploymentAlertsWithoutFailure,
			expectedOutput: "testDeploymentAlertsWithoutFailure.csv",
		},
		"should fail with failing enforcement actions": {
			alerts:         testDeploymentAlertsWithFailure,
			expectedOutput: "testDeploymentAlertsWithFailure.csv",
			shouldFail:     true,
			error:          policy.ErrBreakingPolicies,
		},
	}

	csvPrinter, err := printer.NewTabularPrinterFactory(defaultDeploymentCheckHeaders,
		defaultDeploymentCheckJSONPathExpression).CreatePrinter("csv")
	d.Require().NoError(err)
	d.runOutputTests(cases, csvPrinter, true)
}

func (d *deployCheckTestSuite) TestCheck_LegacyJSONOutput() {
	cases := map[string]outputFormatTest{
		"should render legacy JSON output and return no error with non failing alerts": {
			alerts:         testDeploymentAlertsWithoutFailure,
			expectedOutput: "testDeploymentAlertsWithoutFailure_legacy.json",
		},
		"should render legacy JSON output and return no error with failing alerts": {
			alerts:         testDeploymentAlertsWithFailure,
			expectedOutput: "testDeploymentAlertsWithFailure_legacy.json",
			shouldFail:     false,
		},
		"should render empty output with empty alerts": {
			alerts:         nil,
			expectedOutput: "empty.json",
		},
	}

	d.runLegacyOutputTests(cases, true)
}

func (d *deployCheckTestSuite) runLegacyOutputTests(cases map[string]outputFormatTest, json bool) {
	for name, c := range cases {
		d.Run(name, func() {
			var out *bytes.Buffer
			conn, closeFunction := d.createGRPCMockDetectionService(c.alerts)
			defer closeFunction()

			deployCheckCmd := d.defaultDeploymentCheckCommand
			deployCheckCmd.env, out = d.createMockEnvironmentWithConn(conn)
			deployCheckCmd.json = json

			err := deployCheckCmd.Check()
			if c.shouldFail {
				d.Require().Error(err)
			} else {
				d.Require().NoError(err)
			}
			expectedOutput, err := os.ReadFile(path.Join("testdata", c.expectedOutput))
			d.Require().NoError(err)
			d.Assert().Equal(string(expectedOutput), out.String())
		})
	}
}

func (d *deployCheckTestSuite) runOutputTests(cases map[string]outputFormatTest, printer printer.ObjectPrinter,
	standardizedFormat bool) {
	const colorTestPrefix = "color_"
	for name, c := range cases {
		d.Run(name, func() {
			deployCheckCmd, out, closeF := d.createDeployCheckCmd(c, printer, standardizedFormat)
			defer closeF()

			d.assertError(deployCheckCmd, c)
			expectedOutput, err := os.ReadFile(path.Join("testdata", c.expectedOutput))
			d.Require().NoError(err)
			d.Assert().Equal(string(expectedOutput), out.String())
		})
		d.Run(colorTestPrefix+name, func() {
			if runtime.GOOS == "windows" {
				d.T().Skip("Windows has different color sequences than Linux/Mac.")
			}
			color.NoColor = false
			defer func() { color.NoColor = true }()

			deployCheckCmd, out, closeF := d.createDeployCheckCmd(c, printer, standardizedFormat)
			defer closeF()

			d.assertError(deployCheckCmd, c)
			expectedOutput, err := os.ReadFile(path.Join("testdata", colorTestPrefix+c.expectedOutput))
			d.Require().NoError(err)
			d.Assert().Equal(string(expectedOutput), out.String())
		})
	}
}

func (d *deployCheckTestSuite) assertError(deployCheckCmd deploymentCheckCommand, c outputFormatTest) {
	err := deployCheckCmd.Check()
	if c.shouldFail {
		d.Require().Error(err)
		d.Assert().ErrorIs(err, c.error)
	} else {
		d.Require().NoError(err)
	}
}

func (d *deployCheckTestSuite) createDeployCheckCmd(c outputFormatTest, printer printer.ObjectPrinter, standardizedFormat bool) (deploymentCheckCommand, *bytes.Buffer, func()) {
	conn, closeF := d.createGRPCMockDetectionService(c.alerts)

	deployCheckCmd := d.defaultDeploymentCheckCommand
	deployCheckCmd.printer = printer
	deployCheckCmd.standardizedFormat = standardizedFormat

	var out *bytes.Buffer
	deployCheckCmd.env, out = d.createMockEnvironmentWithConn(conn)
	return deployCheckCmd, out, closeF
}
