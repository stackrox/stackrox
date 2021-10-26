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
	mockEnv := mocks.NewMockEnvironment(gomock.NewController(d.T()))

	_, _, testStdOut, _ := environment.TestIO()
	mockEnv.EXPECT().InputOutput().AnyTimes().Return(environment.DefaultIO())
	mockEnv.EXPECT().GRPCConnection().AnyTimes().Return(conn, nil)

	return mockEnv, testStdOut
}

func (d *deployCheckTestSuite) SetupTest() {
	d.defaultDeploymentCheckCommand = deploymentCheckCommand{
		file:               "testdata/deployment.yaml",
		retryDelay:         3,
		retryCount:         0,
		timeout:            1 * time.Minute,
		printAllViolations: true,
	}
}

func (d *deployCheckTestSuite) TestConstruct() {
	deployCheckCmd := d.defaultDeploymentCheckCommand

	expectedTimeout := 10 * time.Minute

	testCmd := &cobra.Command{Use: "test"}
	testCmd.Flags().Duration("timeout", expectedTimeout, "")

	d.Require().NoError(deployCheckCmd.Construct(nil, testCmd))

	d.Assert().Equal(expectedTimeout, deployCheckCmd.timeout)
}

func (d *deployCheckTestSuite) TestValidate() {
	deployCheckCmd := d.defaultDeploymentCheckCommand

	d.Assert().NoError(deployCheckCmd.Validate())
}

type outputFormatTest struct {
	alerts         []*storage.Alert
	expectedOutput string
	shouldFail     bool
}

func (d *deployCheckTestSuite) TestCheck_LegacyPrettyOutput() {
	cases := map[string]outputFormatTest{
		// There is an issue currently within the output: The policy "policy5" is given two times within an alert
		// but wrongly duplicated within the output.
		"should render legacy pretty output and return no error with non failing alerts": {
			alerts: testDeploymentAlertsWithoutFailure,
			expectedOutput: `

✗ Deployment wordpress failed policy 'policy 7'
- Description:
    ↳ policy 7 for testing
- Rationale:
    ↳ policy 7 for testing
- Remediation:
    ↳ policy 7 for testing
- Violations:
    - testing alert violation message

✗ Deployment wordpress failed policy 'policy 2'
- Description:
    ↳ policy 2 for testing
- Rationale:
    ↳ policy 2 for testing
- Remediation:
    ↳ policy 2 for testing
- Violations:
    - testing multiple alert violation messages 1
    - testing multiple alert violation messages 2
    - testing multiple alert violation messages 3

✗ Deployment wordpress failed policy 'policy 5'
- Description:
    ↳ policy 5 for testing
- Rationale:
    ↳ policy 5 for testing
- Remediation:
    ↳ policy 5 for testing
- Violations:
    - testing alert violation message

✗ Deployment wordpress failed policy 'policy 5'
- Description:
    ↳ policy 5 for testing
- Rationale:
    ↳ policy 5 for testing
- Remediation:
    ↳ policy 5 for testing
- Violations:
    - testing multiple alert violation messages 1
    - testing multiple alert violation messages 2
    - testing multiple alert violation messages 3

✗ Deployment wordpress failed policy 'policy 6'
- Description:
    ↳ policy 6 for testing
- Rationale:
    ↳ policy 6 for testing
- Remediation:
    ↳ policy 6 for testing
- Violations:
    - testing alert violation message

✗ Deployment wordpress failed policy 'policy 1'
- Description:
    ↳ policy 1 for testing
- Rationale:
    ↳ policy 1 for testing
- Remediation:
    ↳ policy 1 for testing
- Violations:
    - testing multiple alert violation messages 1
    - testing multiple alert violation messages 2
    - testing multiple alert violation messages 3

`,
		},
		"should render legacy pretty output and return an error with failing alerts": {
			alerts: testDeploymentAlertsWithFailure,
			expectedOutput: `

✗ Deployment wordpress failed policy 'policy 7'
- Description:
    ↳ policy 7 for testing
- Rationale:
    ↳ policy 7 for testing
- Remediation:
    ↳ policy 7 for testing
- Violations:
    - testing alert violation message

✗ Deployment wordpress failed policy 'policy 2'
- Description:
    ↳ policy 2 for testing
- Rationale:
    ↳ policy 2 for testing
- Remediation:
    ↳ policy 2 for testing
- Violations:
    - testing multiple alert violation messages 1
    - testing multiple alert violation messages 2
    - testing multiple alert violation messages 3

✗ Deployment wordpress failed policy 'policy 5'
- Description:
    ↳ policy 5 for testing
- Rationale:
    ↳ policy 5 for testing
- Remediation:
    ↳ policy 5 for testing
- Violations:
    - testing alert violation message

✗ Deployment wordpress failed policy 'policy 5'
- Description:
    ↳ policy 5 for testing
- Rationale:
    ↳ policy 5 for testing
- Remediation:
    ↳ policy 5 for testing
- Violations:
    - testing multiple alert violation messages 1
    - testing multiple alert violation messages 2
    - testing multiple alert violation messages 3

✗ Deployment wordpress failed policy 'policy 6'
- Description:
    ↳ policy 6 for testing
- Rationale:
    ↳ policy 6 for testing
- Remediation:
    ↳ policy 6 for testing
- Violations:
    - testing alert violation message

✗ Deployment wordpress failed policy 'policy 4' (policy enforcement caused failure)
- Description:
    ↳ policy 4 for testing
- Rationale:
    ↳ policy 4 for testing
- Remediation:
    ↳ policy 4 for testing
- Violations:
    - testing multiple alert violation messages 1
    - testing multiple alert violation messages 2
    - testing multiple alert violation messages 3

`,
			shouldFail: true,
		},
		"should render empty output with empty alerts": {
			alerts: nil,
			expectedOutput: `✔ The scanned resources passed all policies
`,
		},
	}

	d.runLegacyOutputTests(cases, false)
}

func (d *deployCheckTestSuite) TestCheck_LegacyJSONOutput() {
	cases := map[string]outputFormatTest{
		"should render legacy JSON output and return no error with non failing alerts": {
			alerts: testDeploymentAlertsWithoutFailure,
			expectedOutput: `{
  "alerts": [
    {
      "policy": {
        "id": "policy7",
        "name": "policy 7",
        "description": "policy 7 for testing",
        "rationale": "policy 7 for testing",
        "remediation": "policy 7 for testing",
        "severity": "LOW_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing alert violation message"
        }
      ]
    },
    {
      "policy": {
        "id": "policy2",
        "name": "policy 2",
        "description": "policy 2 for testing",
        "rationale": "policy 2 for testing",
        "remediation": "policy 2 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing multiple alert violation messages 1"
        },
        {
          "message": "testing multiple alert violation messages 2"
        },
        {
          "message": "testing multiple alert violation messages 3"
        }
      ]
    },
    {
      "policy": {
        "id": "policy5",
        "name": "policy 5",
        "description": "policy 5 for testing",
        "rationale": "policy 5 for testing",
        "remediation": "policy 5 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing alert violation message"
        }
      ]
    },
    {
      "policy": {
        "id": "policy5",
        "name": "policy 5",
        "description": "policy 5 for testing",
        "rationale": "policy 5 for testing",
        "remediation": "policy 5 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing multiple alert violation messages 1"
        },
        {
          "message": "testing multiple alert violation messages 2"
        },
        {
          "message": "testing multiple alert violation messages 3"
        }
      ]
    },
    {
      "policy": {
        "id": "policy6",
        "name": "policy 6",
        "description": "policy 6 for testing",
        "rationale": "policy 6 for testing",
        "remediation": "policy 6 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing alert violation message"
        }
      ]
    },
    {
      "policy": {
        "id": "policy1",
        "name": "policy 1",
        "description": "policy 1 for testing",
        "rationale": "policy 1 for testing",
        "remediation": "policy 1 for testing",
        "severity": "CRITICAL_SEVERITY",
        "enforcementActions": [
          "FAIL_BUILD_ENFORCEMENT"
        ]
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing multiple alert violation messages 1"
        },
        {
          "message": "testing multiple alert violation messages 2"
        },
        {
          "message": "testing multiple alert violation messages 3"
        }
      ]
    }
  ]
}
`,
		},
		"should render legacy JSON output and return no error with failing alerts": {
			alerts: testDeploymentAlertsWithFailure,
			expectedOutput: `{
  "alerts": [
    {
      "policy": {
        "id": "policy7",
        "name": "policy 7",
        "description": "policy 7 for testing",
        "rationale": "policy 7 for testing",
        "remediation": "policy 7 for testing",
        "severity": "LOW_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing alert violation message"
        }
      ]
    },
    {
      "policy": {
        "id": "policy2",
        "name": "policy 2",
        "description": "policy 2 for testing",
        "rationale": "policy 2 for testing",
        "remediation": "policy 2 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing multiple alert violation messages 1"
        },
        {
          "message": "testing multiple alert violation messages 2"
        },
        {
          "message": "testing multiple alert violation messages 3"
        }
      ]
    },
    {
      "policy": {
        "id": "policy5",
        "name": "policy 5",
        "description": "policy 5 for testing",
        "rationale": "policy 5 for testing",
        "remediation": "policy 5 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing alert violation message"
        }
      ]
    },
    {
      "policy": {
        "id": "policy5",
        "name": "policy 5",
        "description": "policy 5 for testing",
        "rationale": "policy 5 for testing",
        "remediation": "policy 5 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing multiple alert violation messages 1"
        },
        {
          "message": "testing multiple alert violation messages 2"
        },
        {
          "message": "testing multiple alert violation messages 3"
        }
      ]
    },
    {
      "policy": {
        "id": "policy6",
        "name": "policy 6",
        "description": "policy 6 for testing",
        "rationale": "policy 6 for testing",
        "remediation": "policy 6 for testing",
        "severity": "MEDIUM_SEVERITY"
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing alert violation message"
        }
      ]
    },
    {
      "policy": {
        "id": "policy4",
        "name": "policy 4",
        "description": "policy 4 for testing",
        "rationale": "policy 4 for testing",
        "remediation": "policy 4 for testing",
        "severity": "HIGH_SEVERITY",
        "enforcementActions": [
          "SCALE_TO_ZERO_ENFORCEMENT"
        ]
      },
      "deployment": {
        "name": "wordpress",
        "type": "Deployment"
      },
      "violations": [
        {
          "message": "testing multiple alert violation messages 1"
        },
        {
          "message": "testing multiple alert violation messages 2"
        },
        {
          "message": "testing multiple alert violation messages 3"
        }
      ]
    }
  ]
}
`,
			shouldFail: false,
		},
		"should render empty output with empty alerts": {
			alerts: nil,
			expectedOutput: `{

}
`,
		},
	}

	d.runLegacyOutputTests(cases, true)
}

func (d *deployCheckTestSuite) runLegacyOutputTests(cases map[string]outputFormatTest, json bool) {
	for name, c := range cases {
		d.Run(name, func() {
			//var out *bytes.Buffer
			conn, closeFunction := d.createGRPCMockDetectionService(c.alerts)
			defer closeFunction()

			deployCheckCmd := d.defaultDeploymentCheckCommand
			deployCheckCmd.env, _ = d.createMockEnvironmentWithConn(conn)
			deployCheckCmd.json = json

			err := deployCheckCmd.Check()
			if c.shouldFail {
				d.Require().Error(err)
			} else {
				d.Require().NoError(err)
			}
			//d.Assert().Equal(c.expectedOutput, out.String())
		})
	}
}
