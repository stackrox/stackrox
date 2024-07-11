package create

import (
	"testing"

	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAccessScope_Failures(t *testing.T) {
	cases := map[string]struct {
		args   []string
		errOut string
	}{
		"missing name flag": {
			args: []string{
				"access-scope",
				"--description=some-description",
			},
			errOut: `Error: required flag(s) "name" not set
`,
		},
		"invalid operator in label selector": {
			args: []string{
				"access-scope",
				"--name=some-name",
				"--cluster-label-selector=key=some-key;operator=WRONG;values=some-value",
			},
		},
		"invalid label selector flag value": {
			args: []string{
				"access-scope",
				"--name=some-name",
				"--cluster-label-selector=key=some-key,operator=WRONG,values=some-value",
			},
			errOut: `Error: invalid argument "key=some-key,operator=WRONG,values=some-value" for "--cluster-label-selector" flag: key=some-key,operator=WRONG,values=some-value must either be formatted as key=v;operator=v or key=v;operator=v;values=v
`,
		},
		"invalid label selector key value": {
			args: []string{
				"access-scope",
				"--name=some-name",
				"--cluster-label-selector=key:some-key;operator=IN;values=some-value",
			},
			errOut: `Error: invalid argument "key:some-key;operator=IN;values=some-value" for "--cluster-label-selector" flag: key:some-key must specify key=value
`,
		},
		"invalid included objects flag value": {
			args: []string{
				"access-scope",
				"--name=some-name",
				"--included=cluster=namespace=",
			},
			errOut: `Error: invalid argument "cluster=namespace=" for "--included" flag: cluster=namespace= must be either formatted as key or as key=value pair
`,
		},
		"invalid key value pair in label selector": {
			args: []string{
				"access-scope",
				"--name=some-name",
				"--cluster-label-selector=something=somewhere;here=there",
			},
			errOut: `Error: invalid argument "something=somewhere;here=there" for "--cluster-label-selector" flag: something=somewhere must specify either key, operator, values
`,
		},
		"invalid access scope": {
			args: []string{
				"access-scope",
				"--name=some-name",
				"--description=some-description",
				"--included=clusterA",
				"--cluster-label-selector=key=some-key;operator=EXISTS;values=some-value",
			},
			errOut: `Error: validating access scope: 1 error occurred:
	* values: Invalid value: []string{"some-value"}: values set must be empty for exists and does not exist


`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			env, out, errOut := mocks.NewEnvWithConn(nil, t)
			cmd := Command(env)
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
		"access-scope",
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
	cmd := Command(env)
	cmd.SetArgs(args)
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	err := cmd.Execute()
	assert.NoError(t, err)

	assert.Empty(t, errOut)
	assert.Equal(t, expectedYAML, out.String())
}

func TestAccessScope_WriteToK8sObject(t *testing.T) {
	cases := map[string]struct {
		secret                 string
		configMap              string
		shouldWriteToK8sObject bool
	}{
		"no flag set should not write to k8s object": {},
		"config map flag set should write to k8s object": {
			configMap:              "something",
			shouldWriteToK8sObject: true,
		},
		"secret flag set should write to k8s object": {
			secret:                 "something",
			shouldWriteToK8sObject: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			env, _, _ := mocks.NewEnvWithConn(nil, t)
			cmd := Command(env)
			if c.configMap != "" {
				require.NoError(t, cmd.Flags().Set(k8sobject.ConfigMapFlag, c.configMap))
			}
			if c.secret != "" {
				require.NoError(t, cmd.Flags().Set(k8sobject.SecretFlag, c.secret))
			}

			accessScopeCmd := accessScopeCmd{}
			err := accessScopeCmd.Construct(cmd)
			require.NoError(t, err)
			assert.Equal(t, c.shouldWriteToK8sObject, accessScopeCmd.configMap != "" || accessScopeCmd.secret != "")
		})
	}
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
