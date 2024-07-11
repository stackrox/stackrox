package create

import (
	"testing"

	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateNotifier_Failures(t *testing.T) {
	cases := map[string]struct {
		args   []string
		errOut string
		err    error
	}{
		"missing name flag": {
			args:   []string{"notifier", "generic"},
			errOut: "Error: required flag(s) \"name\", \"webhook-endpoint\" not set\n",
		},
		"splunk flags group": {
			args: []string{"notifier", "splunk",
				"--name=some-name",
				"--splunk-token=token",
			},
			errOut: `Error: required flag(s) "splunk-endpoint" not set
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

			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
			}

			if c.errOut != "" {
				assert.Equal(t, c.errOut, errOut.String())
			}
		})
	}
}

func TestCreateNotifier_Success(t *testing.T) {
	cases := map[string]struct {
		args         []string
		expectedYAML string
	}{
		"only name": {
			args: []string{"notifier", "generic",
				"--name=some-name",
				"--webhook-endpoint=some-endpoint",
			},
			expectedYAML: `name: some-name
generic:
    endpoint: some-endpoint
`,
		},
		"with headers": {
			args: []string{"notifier", "generic",
				"--name=some-name",
				"--webhook-endpoint=some-endpoint",
				"--headers=k2=v2,k1=v1",
			},
			expectedYAML: `name: some-name
generic:
    endpoint: some-endpoint
    headers:
        - key: k1
          value: v1
        - key: k2
          value: v2
`,
		},
		"with splunk types": {
			args: []string{"notifier", "splunk",
				"--name=some-name",
				"--splunk-endpoint=some-endpoint",
				"--splunk-token=some-token",
				"--source-types=k2=v2,k1=v1",
			},
			expectedYAML: `name: some-name
splunk:
    token: some-token
    endpoint: some-endpoint
    sourceTypes:
        - key: k1
          sourceType: v1
        - key: k2
          sourceType: v2
`,
		}}

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

func TestNotifier_WriteToK8sObject(t *testing.T) {
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

			nc := notifierCmd{}
			require.NoError(t, nc.construct(cmd))
			assert.Equal(t, c.shouldWriteToK8sObject, nc.configMap != "" || nc.secret != "")
		})
	}
}
