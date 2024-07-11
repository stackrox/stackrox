package create

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePermissionSet_Failures(t *testing.T) {
	cases := map[string]struct {
		args   []string
		errOut string
		err    error
	}{
		"missing name flag": {
			args: []string{
				"permission-set",
				`--resource-with-access="Access=READ_ACCESS"`,
			},
			errOut: `Error: if any flags in the group [name resource-with-access] are set they must all be set; missing [name]
`,
		},
		"missing resource-with-access flag": {
			args: []string{
				"permission-set",
				"--name=some-name",
			},
			errOut: `Error: if any flags in the group [name resource-with-access] are set they must all be set; missing [resource-with-access]
`,
		},
		"invalid access specified in resource-with-access flag": {
			args: []string{
				"permission-set",
				"--name=some-name",
				`--resource-with-access=Access=ReadAccess,Admin=READ_WRITE_ACCESS,Policy=none_access`,
			},
			err: errox.InvalidArgs,
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

			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
			}

			if c.errOut != "" {
				assert.Equal(t, c.errOut, errOut.String())
			}
		})
	}
}

func TestCreatePermissionSet_Success(t *testing.T) {
	cases := map[string]struct {
		args         []string
		expectedYAML string
	}{
		"with description set": {
			args: []string{
				"permission-set",
				"--name=some-name",
				"--description=some-description",
				`--resource-with-access=Access=READ_ACCESS,Admin=READ_WRITE_ACCESS`,
			},
			expectedYAML: `name: some-name
description: some-description
resources:
    - resource: Access
      access: READ_ACCESS
    - resource: Admin
      access: READ_WRITE_ACCESS
`,
		},
		"without description set": {
			args: []string{
				"permission-set",
				"--name=some-name",
				`--resource-with-access=Access=READ_ACCESS,Admin=READ_WRITE_ACCESS`,
			},
			expectedYAML: `name: some-name
resources:
    - resource: Access
      access: READ_ACCESS
    - resource: Admin
      access: READ_WRITE_ACCESS
`,
		},
		"with lowercase resource": {
			args: []string{
				"permission-set",
				"--name=some-name",
				"--resource-with-access=Access=read_access",
				"--resource-with-access=Admin=read_write_access",
			},
			expectedYAML: `name: some-name
resources:
    - resource: Access
      access: READ_ACCESS
    - resource: Admin
      access: READ_WRITE_ACCESS
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

			assert.Empty(t, errOut)
			assert.Equal(t, c.expectedYAML, out.String())
		})
	}
}

func TestPermissionSet_WriteToK8sObject(t *testing.T) {
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

			permissionSetCmd := permissionSetCmd{}
			err := permissionSetCmd.Construct(cmd)
			require.NoError(t, err)
			assert.Equal(t, c.shouldWriteToK8sObject, permissionSetCmd.configMap != "" || permissionSetCmd.secret != "")
		})
	}
}
