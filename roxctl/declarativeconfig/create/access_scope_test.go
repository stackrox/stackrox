package create

import (
	"testing"

	"github.com/stackrox/rox/roxctl/common/mocks"
	"github.com/stretchr/testify/assert"
)

func TestCreateAccessScope_Failures(t *testing.T) {
	cases := map[string]struct {
		args   []string
		errOut string
	}{
		"missing name flag": {
			args: []string{
				"--description=some-description",
			},
			errOut: `Error: required flag(s) "name" not set
`,
		},
		"invalid operator in label selector": {
			args: []string{
				"--name=some-name",
				"--cluster-label-selector=key=some-key;operator=WRONG;values=some-value",
			},
		},
		"invalid label selector flag value": {
			args: []string{
				"--name=some-name",
				"--cluster-label-selector=key=some-key,operator=WRONG,values=some-value",
			},
			errOut: `Error: invalid argument "key=some-key,operator=WRONG,values=some-value" for "--cluster-label-selector" flag: key=some-key,operator=WRONG,values=some-value must either be formatted as key=v;operator=v or key=v;operator=v;values=v
`,
		},
		"invalid label selector key value": {
			args: []string{
				"--name=some-name",
				"--cluster-label-selector=key:some-key;operator=IN;values=some-value",
			},
			errOut: `Error: invalid argument "key:some-key;operator=IN;values=some-value" for "--cluster-label-selector" flag: key:some-key must specify key=value
`,
		},
		"invalid included objects flag value": {
			args: []string{
				"--name=some-name",
				"--included=cluster=namespace=",
			},
			errOut: `Error: invalid argument "cluster=namespace=" for "--included" flag: cluster=namespace= must be either formatted as key or as key=value pair
`,
		},
		"invalid key value pair in label selector": {
			args: []string{
				"--name=some-name",
				"--cluster-label-selector=something=somewhere;here=there",
			},
			errOut: `Error: invalid argument "something=somewhere;here=there" for "--cluster-label-selector" flag: something=somewhere must specify either key, operator, values
`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			env, out, errOut := mocks.NewEnvWithConn(nil, t)
			cmd := accessScopeCommand(env)
			cmd.SetArgs(c.args)
			cmd.SetOut(out)
			cmd.SetErr(errOut)

			err := cmd.Execute()
			assert.Error(t, err)
			if c.errOut != "" {
				assert.Equal(t, c.errOut, errOut.String())
			}
		})
	}
}

func TestCreateAccessScope_Success(t *testing.T) {
	args := []string{
		"--name=some-name",
		"--description=some-description",
		"--included=clusterA",
		"--included=clusterB=namespaceA,namespaceB",
		"--cluster-label-selector=key=some-key;operator=IN;values=some-value,another-value",
		"--cluster-label-selector=key=some-key;operator=EXISTS",
		"--namespace-label-selector=key=some-key;operator=IN;values=some-value",
		"--namespace-label-selector=key=some-key;operator=EXISTS",
	}

	expectedYAML := `name: some-name
description: some-description
rules:
    included:
        - cluster: clusterA
        - cluster: clusterB
          namespaces:
            - namespaceA
            - namespaceB
    clusterLabelSelectors:
        - requirements:
            - key: some-key
              operator: IN
              values:
                - some-value
                - another-value
            - key: some-key
              operator: EXISTS
    namespaceLabelSelectors:
        - requirements:
            - key: some-key
              operator: IN
              values:
                - some-value
            - key: some-key
              operator: EXISTS
`

	env, out, errOut := mocks.NewEnvWithConn(nil, t)
	cmd := accessScopeCommand(env)
	cmd.SetArgs(args)
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	err := cmd.Execute()
	assert.NoError(t, err)

	assert.Empty(t, errOut)
	assert.Equal(t, expectedYAML, out.String())
}

func FuzzRetrieveRequirement(f *testing.F) {
	args := []string{"key=some-key;operator=IN;values=some-value,another-value", "key=some-key;operator=EXISTS"}
	for _, arg := range args {
		f.Add(arg)
	}

	f.Fuzz(func(t *testing.T, s string) {
		assert.NotPanics(t, func() {
			_, _ = retrieveRequirement(s)
		})
	})
}
