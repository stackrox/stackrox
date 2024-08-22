package report

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	updateFlag = flag.Bool("update", false, "update .golden files")

	imageAlertOne = storage.Alert{
		Policy: &storage.Policy{
			Name:        "CI Test Policy One",
			Description: "CI policy one that is used for tests",
			Severity:    storage.Severity_CRITICAL_SEVERITY,
			Rationale:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent nec orci bibendum sapien suscipit maximus. Quisque dapibus accumsan tempor. Vivamus at justo tellus. Vivamus vel sem vel mauris cursus ullamcorper.",
			Remediation: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vivamus cursus convallis lacinia.",
			EnforcementActions: []storage.EnforcementAction{
				storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
			},
		},
		Violations: []*storage.Alert_Violation{
			{
				Message: "This is awesome",
			},
			{
				Message: "This is more awesome",
			},
		},
	}

	imageAlertTwo = storage.Alert{
		Policy: &storage.Policy{
			Name:        "CI Test Policy Two",
			Description: "CI policy two that is used for tests",
			Severity:    storage.Severity_MEDIUM_SEVERITY,
			Rationale:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean eleifend ac purus id vehicula. Vivamus malesuada eros at malesuada scelerisque. Praesent pellentesque ipsum mauris, eu tempus diam interdum quis.",
			Remediation: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin nec vehicula magna.",
		},
		Violations: []*storage.Alert_Violation{
			{
				Message: "This is cool",
			},
			{
				Message: "This is more cool",
			},
		},
	}
	imageAlertThree = storage.Alert{
		Policy: &storage.Policy{
			Name:        "CI Test Policy Three",
			Description: "CI policy three that is used for tests",
			Severity:    storage.Severity_MEDIUM_SEVERITY,
			Rationale:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean eleifend ac purus id vehicula. Vivamus malesuada eros at malesuada scelerisque. Praesent pellentesque ipsum mauris, eu tempus diam interdum quis.",
			Remediation: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin nec vehicula magna.",
		},
		Violations: []*storage.Alert_Violation{
			{Message: "This is cool"},
			{Message: "This is more cool"},
			{Message: "This is another violation"},
			{Message: "This is a lot of violations"},
			{Message: "This would be neat if I could come up with"},
			{Message: "A lot of unique violations"},
			{Message: "This might make the code reviewers laugh"},
			{Message: "But I have run out of words"},
			{Message: "nine"},
			{Message: "ten"},
			{Message: "eleven"},
			{Message: "twelve"},
			{Message: "thirteen"},
			{Message: "fourteen"},
			{Message: "fifteen"},
			{Message: "sixteen"},
			{Message: "seventeen"},
			{Message: "eighteen"},
			{Message: "ninteen"},
			{Message: "twenty"},
			{Message: "twenty one"},
			{Message: "twenty two"},
			{Message: "twenty three"},
			{Message: "twenty four"},
		},
	}

	deploymentAlertOne = storage.Alert{
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{
			Name: "deployment1",
		}},
		Policy: &storage.Policy{
			Name:        "CI Test Policy One",
			Description: "CI policy one that is used for tests",
			Severity:    storage.Severity_CRITICAL_SEVERITY,
			Rationale:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent nec orci bibendum sapien suscipit maximus. Quisque dapibus accumsan tempor. Vivamus at justo tellus. Vivamus vel sem vel mauris cursus ullamcorper.",
			Remediation: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vivamus cursus convallis lacinia.",
			EnforcementActions: []storage.EnforcementAction{
				storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
			},
		},
		Violations: []*storage.Alert_Violation{
			{
				Message: "This is awesome",
			},
			{
				Message: "This is more awesome",
			},
		},
	}

	deploymentAlertTwo = storage.Alert{
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{
			Name: "deployment2",
		}},
		Policy: &storage.Policy{
			Name:        "CI Test Policy Two",
			Description: "CI policy two that is used for tests",
			Severity:    storage.Severity_MEDIUM_SEVERITY,
			Rationale:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean eleifend ac purus id vehicula. Vivamus malesuada eros at malesuada scelerisque. Praesent pellentesque ipsum mauris, eu tempus diam interdum quis.",
			Remediation: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin nec vehicula magna.",
		},
		Violations: []*storage.Alert_Violation{
			{
				Message: "This is cool",
			},
			{
				Message: "This is more cool",
			},
		},
	}

	remarkOne = v1.DeployDetectionRemark{
		Name:                   "deployment1",
		PermissionLevel:        storage.PermissionLevel_CLUSTER_ADMIN.String(),
		AppliedNetworkPolicies: []string{"Policy1, Policy2"},
	}

	remarkTwo = v1.DeployDetectionRemark{
		Name:                   "deployment2",
		PermissionLevel:        storage.PermissionLevel_NONE.String(),
		AppliedNetworkPolicies: nil,
	}
)

func TestReport(t *testing.T) {
	tests := []struct {
		name               string
		resourceType       string
		resourceName       string
		alerts             []*storage.Alert
		goldenFile         string
		printAllViolations bool
	}{
		{
			name:         "nil image alerts",
			resourceType: "Image",
			resourceName: "nginx",
			alerts:       nil,
			goldenFile:   "testdata/passed.txt",
		},
		{
			name:         "empty image alerts",
			resourceType: "Image",
			resourceName: "nginx",
			alerts:       []*storage.Alert{},
			goldenFile:   "testdata/passed.txt",
		},
		{
			name:         "single image alert",
			resourceType: "Image",
			resourceName: "nginx",
			alerts:       []*storage.Alert{&imageAlertOne},
			goldenFile:   "testdata/one-image.txt",
		},
		{
			name:         "multiple image alerts",
			resourceType: "Image",
			resourceName: "nginx",
			alerts:       []*storage.Alert{&imageAlertTwo},
			goldenFile:   "testdata/two-images.txt",
		},
		{
			name:         "nil deployment alerts",
			resourceType: "Deployment",
			alerts:       nil,
			goldenFile:   "testdata/passed.txt",
		},
		{
			name:         "empty deployment alerts",
			resourceType: "Deployment",
			alerts:       []*storage.Alert{},
			goldenFile:   "testdata/passed.txt",
		},
		{
			name:         "single deployment alert",
			resourceType: "Deployment",
			alerts:       []*storage.Alert{&deploymentAlertOne},
			goldenFile:   "testdata/one-deployment.txt",
		},
		{
			name:         "multiple deployment alerts",
			resourceType: "Deployment",
			alerts:       []*storage.Alert{&deploymentAlertOne, &deploymentAlertTwo},
			goldenFile:   "testdata/two-deployments.txt",
		},
		{
			name:         "hit violation cutoff",
			resourceType: "Image",
			resourceName: "nginx",
			alerts:       []*storage.Alert{&imageAlertThree},
			goldenFile:   "testdata/many-violations.txt",
		}, {
			name:               "hit violation cutoff but print them all",
			resourceType:       "Image",
			resourceName:       "nginx",
			alerts:             []*storage.Alert{&imageAlertThree},
			goldenFile:         "testdata/many-violations-all-printed.txt",
			printAllViolations: true,
		},
	}

	for index, test := range tests {
		name := fmt.Sprintf("#%d - %s", index+1, test.name)
		t.Run(name, func(t *testing.T) {
			a := assert.New(t)
			buf := bytes.NewBuffer(nil)
			var enforcementAction storage.EnforcementAction
			switch test.resourceType {
			case "Image":
				enforcementAction = storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT
			case "Deployment":
				enforcementAction = storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT
			default:
				t.Fatalf("Resource type %q is not recognized", test.resourceType)
			}
			a.NoError(PrettyWithResourceName(buf, test.alerts, enforcementAction, test.resourceType, test.resourceName, test.printAllViolations))

			// If the -update flag was passed to go test, update the contents
			// of all golden files.
			if *updateFlag {
				a.NoError(os.WriteFile(test.goldenFile, buf.Bytes(), 0644))
				return
			}

			raw, err := os.ReadFile(test.goldenFile)
			require.Nil(t, err)
			assert.Equal(t, string(raw), buf.String())
		})
	}
}

func TestJSONRemarks(t *testing.T) {
	cases := map[string]struct {
		alerts     []*storage.Alert
		remarks    []*v1.DeployDetectionRemark
		goldenFile string
	}{
		"one alert without remarks": {
			alerts:     []*storage.Alert{&imageAlertOne},
			remarks:    nil,
			goldenFile: "testdata/one-alert.json",
		},
		"multi alerts with empty remarks": {
			alerts:     []*storage.Alert{&imageAlertOne, &imageAlertTwo},
			remarks:    make([]*v1.DeployDetectionRemark, 0),
			goldenFile: "testdata/multi-alerts.json",
		},
		"multi alerts with remarks": {
			alerts:     []*storage.Alert{&imageAlertOne, &imageAlertTwo},
			remarks:    []*v1.DeployDetectionRemark{&remarkOne, &remarkTwo},
			goldenFile: "testdata/multi-alerts-remarks.json",
		},
		"nil alerts with remarks": {
			alerts:     nil,
			remarks:    []*v1.DeployDetectionRemark{&remarkOne, &remarkTwo},
			goldenFile: "testdata/nil-alerts-multi-remarks.json",
		},
		"empty alerts with remarks": {
			alerts:     make([]*storage.Alert, 0),
			remarks:    []*v1.DeployDetectionRemark{&remarkOne, &remarkTwo},
			goldenFile: "testdata/empty-alerts-multi-remarks.json",
		},
		"empty alerts with empty remarks": {
			alerts:     make([]*storage.Alert, 0),
			remarks:    make([]*v1.DeployDetectionRemark, 0),
			goldenFile: "testdata/empty-alerts-remarks.json",
		},
		"nil alerts with nil remarks": {
			alerts:     nil,
			remarks:    nil,
			goldenFile: "testdata/nil-alerts-remarks.json",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			a := assert.New(t)
			buf := bytes.NewBuffer(nil)
			a.NoError(JSONWithRemarks(buf, c.alerts, c.remarks))

			// If the -update flag was passed to go test, update the contents
			// of all golden files.
			if *updateFlag {
				a.NoError(os.WriteFile(c.goldenFile, buf.Bytes(), 0644))
				return
			}

			raw, err := os.ReadFile(c.goldenFile)
			a.NoError(err)
			a.JSONEq(buf.String(), string(raw))
		})
	}

	errorCases := map[string]struct {
		alerts        []*storage.Alert
		remarks       []*v1.DeployDetectionRemark
		writeableLen  int
		expectedError error
	}{
		"multi alerts with remarks, payload write fails": {
			alerts:        []*storage.Alert{&imageAlertOne, &imageAlertTwo},
			remarks:       []*v1.DeployDetectionRemark{&remarkOne, &remarkTwo},
			writeableLen:  100,
			expectedError: errors.Wrap(capacityExhaustedErr, "could not marshal alerts: failed to write JSON"),
		},
		"multi alerts with remarks, trailing newline write fails": {
			alerts:        []*storage.Alert{&imageAlertOne, &imageAlertTwo},
			remarks:       []*v1.DeployDetectionRemark{&remarkOne, &remarkTwo},
			writeableLen:  1704,
			expectedError: errors.Wrap(capacityExhaustedErr, "could not write final newline for alerts"),
		},
	}

	for name, tc := range errorCases {
		t.Run(name, func(t *testing.T) {
			writer := newTestWriter(tc.writeableLen)
			err := JSONWithRemarks(writer, tc.alerts, tc.remarks)
			require.Error(t, err)
			assert.EqualError(t, err, tc.expectedError.Error())
		})
	}
}

func TestJSON(t *testing.T) {
	testCases := []struct {
		name           string
		alerts         []*storage.Alert
		writeableLen   int
		expectedError  error
		expectedOutput string
	}{
		{
			name:          "One alert, write succeeds",
			alerts:        []*storage.Alert{&imageAlertOne},
			writeableLen:  0,
			expectedError: nil,
			expectedOutput: `{
  "alerts":[
    {
      "policy":{
        "name":"CI Test Policy One",
        "description":"CI policy one that is used for tests",
        "rationale":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent nec orci bibendum sapien suscipit maximus. Quisque dapibus accumsan tempor. Vivamus at justo tellus. Vivamus vel sem vel mauris cursus ullamcorper.",
        "remediation":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vivamus cursus convallis lacinia.",
        "severity":"CRITICAL_SEVERITY",
        "enforcementActions":["FAIL_BUILD_ENFORCEMENT"]
      },
      "violations":[
        {"message":"This is awesome"},
        {"message":"This is more awesome"}
      ]
    }
  ]
}`,
		},
		{
			name:          "One alert, payload write fails",
			alerts:        []*storage.Alert{&imageAlertOne},
			writeableLen:  100,
			expectedError: errors.Wrap(capacityExhaustedErr, "could not marshal alerts: failed to write JSON"),
		},
		{
			name:          "One alert, trailing newline write fails",
			alerts:        []*storage.Alert{&imageAlertOne},
			writeableLen:  788,
			expectedError: errors.Wrap(capacityExhaustedErr, "could not write alerts"),
		},
		{
			name:          "Two alerts, write succeeds",
			alerts:        []*storage.Alert{&imageAlertOne, &deploymentAlertTwo},
			writeableLen:  0,
			expectedError: nil,
			expectedOutput: `{
  "alerts":[
    {
      "policy":{
        "name":"CI Test Policy One",
        "description":"CI policy one that is used for tests",
        "rationale":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent nec orci bibendum sapien suscipit maximus. Quisque dapibus accumsan tempor. Vivamus at justo tellus. Vivamus vel sem vel mauris cursus ullamcorper.",
        "remediation":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vivamus cursus convallis lacinia.",
        "severity":"CRITICAL_SEVERITY",
        "enforcementActions":["FAIL_BUILD_ENFORCEMENT"]
      },
      "violations":[
        {"message":"This is awesome"},
        {"message":"This is more awesome"}
      ]
    },
    {
      "deployment":{"name":"deployment2"},
      "policy":{
        "name":"CI Test Policy Two",
        "description":"CI policy two that is used for tests",
        "rationale":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean eleifend ac purus id vehicula. Vivamus malesuada eros at malesuada scelerisque. Praesent pellentesque ipsum mauris, eu tempus diam interdum quis.",
        "remediation":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin nec vehicula magna.",
        "severity":"MEDIUM_SEVERITY"
      },
      "violations":[
        {"message":"This is cool"},
        {"message":"This is more cool"}
      ]
    }
  ]
}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writer := newTestWriter(tc.writeableLen)
			err := JSON(writer, tc.alerts)
			if tc.expectedError != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, tc.expectedOutput, writer.String())
			}
		})
	}
}

type testWriter struct {
	writer strings.Builder
	cap    int
}

func newTestWriter(capacity int) *testWriter {
	if capacity == 0 {
		// 0 should mean "infinite" capacity.
		// Ensure the writer should have enough capacity.
		capacity = 1024 * 1024 * 1024 * 2
	}
	return &testWriter{
		writer: strings.Builder{},
		cap:    capacity,
	}
}

var (
	capacityExhaustedErrorText = "could not write, capacity exhausted"
	capacityExhaustedErr       = errors.New(capacityExhaustedErrorText)
)

func (w *testWriter) Write(p []byte) (n int, err error) {
	written := 0
	for _, b := range p {
		if w.cap == 0 {
			return written, capacityExhaustedErr
		}
		w.writer.WriteByte(b)
		w.cap--
	}
	return written, nil
}

func (w *testWriter) String() string {
	return w.writer.String()
}
