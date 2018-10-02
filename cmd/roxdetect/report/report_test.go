package report

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	updateFlag = flag.Bool("update", false, "update .golden files")

	policyOne = v1.Policy{
		Name:        "CI Test Policy One",
		Description: "CI policy one that is used for tests",
		Severity:    v1.Severity_CRITICAL_SEVERITY,
		Rationale:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent nec orci bibendum sapien suscipit maximus. Quisque dapibus accumsan tempor. Vivamus at justo tellus. Vivamus vel sem vel mauris cursus ullamcorper.",
		Remediation: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vivamus cursus convallis lacinia.",
		Categories:  []string{"Vuln Management", "Security Best Practices"},
	}

	policyTwo = v1.Policy{
		Name:        "CI Test Policy Two",
		Description: "CI policy two that is used for tests",
		Severity:    v1.Severity_MEDIUM_SEVERITY,
		Rationale:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean eleifend ac purus id vehicula. Vivamus malesuada eros at malesuada scelerisque. Praesent pellentesque ipsum mauris, eu tempus diam interdum quis.",
		Remediation: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin nec vehicula magna.",
		Categories:  []string{"Vuln Management", "Security Best Practices"},
	}
)

func TestReport(t *testing.T) {
	tests := []struct {
		name       string
		policies   func() []*v1.Policy
		goldenFile string
	}{
		{
			name: "nil policies",
			policies: func() []*v1.Policy {
				return nil
			},
			goldenFile: "testdata/passed.txt",
		},
		{
			name: "empty policies",
			policies: func() []*v1.Policy {
				return []*v1.Policy{}
			},
			goldenFile: "testdata/passed.txt",
		},
		{
			name: "single policy",
			policies: func() []*v1.Policy {
				return []*v1.Policy{&policyOne}
			},
			goldenFile: "testdata/single-policy.txt",
		},
		{
			name: "two policies",
			policies: func() []*v1.Policy {
				return []*v1.Policy{&policyOne, &policyTwo}
			},
			goldenFile: "testdata/two-policies.txt",
		},
		{
			name: "no description",
			policies: func() []*v1.Policy {
				policyTemp := policyOne
				policyTemp.Description = ""
				return []*v1.Policy{&policyTemp}
			},
			goldenFile: "testdata/no-description.txt",
		},
		{
			name: "no rationale",
			policies: func() []*v1.Policy {
				policyTemp := policyOne
				policyTemp.Rationale = ""
				return []*v1.Policy{&policyTemp}
			},
			goldenFile: "testdata/no-rationale.txt",
		},
		{
			name: "no remediation",
			policies: func() []*v1.Policy {
				policyTemp := policyOne
				policyTemp.Remediation = ""
				return []*v1.Policy{&policyTemp}
			},
			goldenFile: "testdata/no-remediation.txt",
		},
		{
			name: "one category",
			policies: func() []*v1.Policy {
				policyTemp := policyOne
				policyTemp.Categories = []string{"Vuln Management"}
				return []*v1.Policy{&policyTemp}
			},
			goldenFile: "testdata/one-category.txt",
		},
		{
			name: "no categories",
			policies: func() []*v1.Policy {
				policyTemp := policyOne
				policyTemp.Categories = nil
				return []*v1.Policy{&policyTemp}
			},
			goldenFile: "testdata/no-categories.txt",
		},
	}

	for index, test := range tests {
		name := fmt.Sprintf("#%d - %s", index+1, test.name)
		t.Run(name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			Pretty(buf, test.policies())

			// If the -update flag was passed to go test, update the contents
			// of all golden files.
			if *updateFlag {
				ioutil.WriteFile(test.goldenFile, buf.Bytes(), 0644)
				return
			}

			raw, err := ioutil.ReadFile(test.goldenFile)
			require.Nil(t, err)
			assert.Equal(t, string(raw), buf.String())
		})
	}
}
