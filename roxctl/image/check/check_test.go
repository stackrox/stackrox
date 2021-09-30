package check

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_reportCheckResults(t *testing.T) {
	env := environment.NewCLIEnvironment(environment.DiscardIO())
	noFatalAlerts := []*storage.Alert{
		{
			Id: "1234",
			Policy: &storage.Policy{
				Name: "some-policy",
			},
		},
	}
	fatalAlerts := []*storage.Alert{
		{
			Id: "1234",
			Policy: &storage.Policy{
				Name: "some-policy",
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
				},
			},
		},
	}
	tests := []struct {
		name    string
		alerts  []*storage.Alert
		wantErr bool
		icCmd   imageCheckCommand
	}{
		{"legacy-nonJSON-noFatalAlerts", noFatalAlerts, false, imageCheckCommand{json: false, failViolationsWithJSON: false, env: env}},
		{"legacy-nonJSON-fatalAlerts", fatalAlerts, true, imageCheckCommand{json: false, failViolationsWithJSON: false, env: env}},
		{"legacy-JSON-noFatalAlerts", noFatalAlerts, false, imageCheckCommand{json: true, failViolationsWithJSON: false, env: env}},
		{"legacy-JSON-fatalAlerts", fatalAlerts, false, imageCheckCommand{json: true, failViolationsWithJSON: false, env: env}},
		{"fixed-nonJSON-noFatalAlerts", noFatalAlerts, false, imageCheckCommand{json: false, failViolationsWithJSON: true, env: env}},
		{"fixed-nonJSON-fatalAlerts", fatalAlerts, true, imageCheckCommand{json: false, failViolationsWithJSON: true, env: env}},
		{"fixed-JSON-noFatalAlerts", noFatalAlerts, false, imageCheckCommand{json: true, failViolationsWithJSON: true, env: env}},
		{"fixed-JSON-fatalAlerts", fatalAlerts, true, imageCheckCommand{json: true, failViolationsWithJSON: true, env: env}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := tt.icCmd.reportCheckResults(tt.alerts); (err != nil) != tt.wantErr {
				t.Errorf("reportCheckResults() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImageCheckCommand_Validate(t *testing.T) {
	cases := map[string]struct {
		i              imageCheckCommand
		expectedErrOut string
	}{
		"failViolations false and json as well, no output expected": {
			i:              imageCheckCommand{failViolationsWithJSON: false, json: false},
			expectedErrOut: "",
		},
		"failViolations true and json not, warning output expected": {
			i:              imageCheckCommand{failViolationsWithJSON: true, json: false},
			expectedErrOut: "Note: --json-fail-on-policy-violations has no effect when --json is not specified.\n",
		},
		"failViolations true and json as well, no output expected": {
			i:              imageCheckCommand{failViolationsWithJSON: true, json: true},
			expectedErrOut: "",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			io, _, _, errOut := environment.TestIO()
			c.i.env = environment.NewCLIEnvironment(io)
			assert.NoError(t, c.i.Validate())
			assert.Equal(t, errOut.String(), c.expectedErrOut)
		})
	}
}

func TestImageCheckCommand_reportCheckResults_OutputFormat(t *testing.T) {

	cases := map[string]struct {
		i              imageCheckCommand
		alerts         []*storage.Alert
		shouldFail     bool
		expectedOutput string
	}{
		"alert with pretty output format not failing the build": {
			alerts: []*storage.Alert{
				{
					Id: "alert1",
					Policy: &storage.Policy{
						Id:          "policy1",
						Name:        "policy1",
						Description: "policy description 1",
						Rationale:   "policy rationale 1",
						Remediation: "policy remediation 1",
					},
					Violations: []*storage.Alert_Violation{
						{
							Message: "alert violation 1",
						},
					},
				},
			},
			shouldFail: false,
			i: imageCheckCommand{
				image:              "nginx",
				printAllViolations: true,
			},
			expectedOutput: "\n\n✗ Image nginx failed policy 'policy1' \n- Description:\n    ↳ policy description 1\n- Rationale:\n    ↳ policy rationale 1\n- Remediation:\n    ↳ policy remediation 1\n- Violations:\n    - alert violation 1\n\n",
		},
		"alert with json output format not failing the build": {
			alerts: []*storage.Alert{
				{
					Id: "alert1",
					Policy: &storage.Policy{
						Id:          "policy1",
						Name:        "policy1",
						Description: "policy description 1",
						Rationale:   "policy rationale 1",
						Remediation: "policy remediation 1",
					},
					Violations: []*storage.Alert_Violation{
						{
							Message: "alert violation 1",
						},
					},
				},
			},
			shouldFail: false,
			i: imageCheckCommand{
				image:              "nginx",
				json:               true,
				printAllViolations: true,
			},
			expectedOutput: "{\n  \"alerts\": [\n    {\n      \"id\": \"alert1\",\n      \"policy\": {\n        \"id\": \"policy1\",\n        \"name\": \"policy1\",\n        \"description\": \"policy description 1\",\n        \"rationale\": \"policy rationale 1\",\n        \"remediation\": \"policy remediation 1\"\n      },\n      \"violations\": [\n        {\n          \"message\": \"alert violation 1\"\n        }\n      ]\n    }\n  ]\n}\n",
		},
		"alert with pretty output format failing the build": {
			alerts: []*storage.Alert{
				{
					Id: "alert1",
					Policy: &storage.Policy{
						Id:                 "policy1",
						Name:               "policy1",
						Description:        "policy description 1",
						Rationale:          "policy rationale 1",
						Remediation:        "policy remediation 1",
						EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
					},
					Violations: []*storage.Alert_Violation{
						{
							Message: "alert violation 1",
						},
					},
				},
			},
			shouldFail: true,
			i: imageCheckCommand{
				image:              "nginx",
				printAllViolations: true,
			},
			expectedOutput: "\n\n✗ Image nginx failed policy 'policy1' (policy enforcement caused failure)\n- Description:\n    ↳ policy description 1\n- Rationale:\n    ↳ policy rationale 1\n- Remediation:\n    ↳ policy remediation 1\n- Violations:\n    - alert violation 1\n\n",
		},
		"alert with json output format failing the build": {
			alerts: []*storage.Alert{
				{
					Id: "alert1",
					Policy: &storage.Policy{
						Id:                 "policy1",
						Name:               "policy1",
						Description:        "policy description 1",
						Rationale:          "policy rationale 1",
						Remediation:        "policy remediation 1",
						EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
					},
					Violations: []*storage.Alert_Violation{
						{
							Message: "alert violation 1",
						},
					},
				},
			},
			shouldFail: true,
			i: imageCheckCommand{
				image:                  "nginx",
				printAllViolations:     true,
				json:                   true,
				failViolationsWithJSON: true,
			},
			expectedOutput: "{\n  \"alerts\": [\n    {\n      \"id\": \"alert1\",\n      \"policy\": {\n        \"id\": \"policy1\",\n        \"name\": \"policy1\",\n        \"description\": \"policy description 1\",\n        \"rationale\": \"policy rationale 1\",\n        \"remediation\": \"policy remediation 1\",\n        \"enforcementActions\": [\n          \"FAIL_BUILD_ENFORCEMENT\"\n        ]\n      },\n      \"violations\": [\n        {\n          \"message\": \"alert violation 1\"\n        }\n      ]\n    }\n  ]\n}\n",
		},
		"alert with json output format failing the build without fail violations flag": {
			alerts: []*storage.Alert{
				{
					Id: "alert1",
					Policy: &storage.Policy{
						Id:                 "policy1",
						Name:               "policy1",
						Description:        "policy description 1",
						Rationale:          "policy rationale 1",
						Remediation:        "policy remediation 1",
						EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
					},
					Violations: []*storage.Alert_Violation{
						{
							Message: "alert violation 1",
						},
					},
				},
			},
			shouldFail: false,
			i: imageCheckCommand{
				image:              "nginx",
				printAllViolations: true,
				json:               true,
			},
			expectedOutput: "{\n  \"alerts\": [\n    {\n      \"id\": \"alert1\",\n      \"policy\": {\n        \"id\": \"policy1\",\n        \"name\": \"policy1\",\n        \"description\": \"policy description 1\",\n        \"rationale\": \"policy rationale 1\",\n        \"remediation\": \"policy remediation 1\",\n        \"enforcementActions\": [\n          \"FAIL_BUILD_ENFORCEMENT\"\n        ]\n      },\n      \"violations\": [\n        {\n          \"message\": \"alert violation 1\"\n        }\n      ]\n    }\n  ]\n}\n",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			io, _, out, _ := environment.TestIO()
			c.i.env = environment.NewCLIEnvironment(io)
			err := c.i.reportCheckResults(c.alerts)
			if c.shouldFail {
				require.Error(t, err)
				require.Equal(t, "Violated a policy with CI enforcement set", err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, c.expectedOutput, out.String())
		})
	}
}
