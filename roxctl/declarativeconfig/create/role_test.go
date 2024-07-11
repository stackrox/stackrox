package create

import (
	"testing"

	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRoleCommand_Failures(t *testing.T) {
	cases := map[string]struct {
		args   []string
		errOut string
	}{
		"no flag set": {
			args: []string{
				"role",
			},
			errOut: `Error: required flag(s) "access-scope", "name", "permission-set" not set
`,
		},
		"missing name flag": {
			args: []string{
				"role",
				"--access-scope=some-access-scope",
				"--permission-set=some-permission-set",
			},
			errOut: `Error: required flag(s) "name" not set
`,
		},
		"missing access scope flag": {
			args: []string{
				"role",
				"--name=some-name",
				"--permission-set=some-permission-set",
			},
			errOut: `Error: required flag(s) "access-scope" not set
`,
		},
		"missing permission set flag": {
			args: []string{
				"role",
				"--name=some-name",
				"--access-scope=some-access-scope",
			},
			errOut: `Error: required flag(s) "permission-set" not set
`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			env, out, errOut := mocks.NewEnvWithConn(nil, t)
			cmd := Command(env)

			cmd.SetArgs(c.args)
			cmd.SetErr(errOut)
			cmd.SetOut(out)

			err := cmd.Execute()
			assert.Error(t, err)
			assert.Equal(t, c.errOut, errOut.String())
		})
	}
}

func TestCreateRoleCommand_Success(t *testing.T) {
	cases := map[string]struct {
		args         []string
		expectedYAML string
	}{
		"with description set": {
			args: []string{
				"role",
				"--name=some-name",
				"--description=some-description",
				"--access-scope=some-access-scope",
				"--permission-set=some-permission-set",
			},
			expectedYAML: `name: some-name
description: some-description
accessScope: some-access-scope
permissionSet: some-permission-set
`,
		},
		"without description set": {
			args: []string{
				"role",
				"--name=some-name",
				"--access-scope=some-access-scope",
				"--permission-set=some-permission-set",
			},
			expectedYAML: `name: some-name
accessScope: some-access-scope
permissionSet: some-permission-set
`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			env, out, errOut := mocks.NewEnvWithConn(nil, t)
			cmd := Command(env)

			cmd.SetArgs(c.args)
			cmd.SetErr(errOut)
			cmd.SetOut(out)

			err := cmd.Execute()
			assert.NoError(t, err)
			assert.Empty(t, errOut.String())
			assert.Equal(t, c.expectedYAML, out.String())
		})
	}
}

func TestRole_WriteToK8sObject(t *testing.T) {
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

			roleCmd := roleCmd{}
			err := roleCmd.Construct(cmd)
			require.NoError(t, err)
			assert.Equal(t, c.shouldWriteToK8sObject, roleCmd.configMap != "" || roleCmd.secret != "")
		})
	}
}
