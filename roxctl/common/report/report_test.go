package report

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"testing"

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
