package lint

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLintCommand_Failure(t *testing.T) {
	cases := map[string]struct {
		yaml []byte
		err  error
	}{
		"invalid YAML": {
			yaml: []byte(`
X
`),
		},
		"unknown configuration type": {
			yaml: []byte(`name: test-name
description: test-description
policy:
  name: my-policy
  enforcement: DEPLOY_TIME`),
		},
		"invalid configuration during unmarshal": {
			yaml: []byte(`name: test-name
description: test-description
resources:
- resource: a
  access: INVALID_ACCESS
`),
			err: errox.InvalidArgs,
		},
		"invalid configuration during transformation": {
			yaml: []byte(`name: test-name
minimumRole: "None"
uiEndpoint: "localhost:8000"
`),
			err: errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			l := lintCmd{fileContents: [][]byte{c.yaml}}
			err := l.Lint()
			assert.Error(t, err)
			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
			}
		})
	}
}

func TestLintCommand_Success(t *testing.T) {
	cases := map[string]struct {
		yaml []byte
	}{
		"single configuration in file": {
			yaml: []byte(`name: some-name
minimumRole: Analyst
uiEndpoint: localhost:8000
extraUIEndpoints:
    - localhost:9090
groups:
    - key: email
      value: example@example.com
      role: Admin
requiredAttributes:
    - key: org_id
      value: "12345"
    - key: name
      value: some_name
claimMappings:
    - path: org_id
      name: super_cool_claim
    - path: republic
      name: far_away
oidc:
    issuer: sample.issuer.com
    mode: auto
    clientID: CLIENT_ID
    clientSecret: CLIENT_SECRET
`),
		},
		"multiple configurations in file": {
			yaml: []byte(`- name: some-name
  minimumRole: Analyst
  uiEndpoint: localhost:8000
  extraUIEndpoints:
      - localhost:9090
  groups:
      - key: email
        value: example@example.com
        role: Admin
  requiredAttributes:
      - key: org_id
        value: "12345"
      - key: name
        value: some_name
  claimMappings:
      - path: org_id
        name: super_cool_claim
      - path: republic
        name: far_away
  oidc:
      issuer: sample.issuer.com
      mode: auto
      clientID: CLIENT_ID
      clientSecret: CLIENT_SECRET
- name: some-name
  description: some-description
  resources:
    - resource: Access
      access: READ_ACCESS
    - resource: Admin
      access: READ_WRITE_ACCESS
`),
		},
		"multiple configurations in file with YAML delimiter": {
			yaml: []byte(`name: some-name
minimumRole: Analyst
uiEndpoint: localhost:8000
extraUIEndpoints:
    - localhost:9090
groups:
    - key: email
      value: example@example.com
      role: Admin
requiredAttributes:
    - key: org_id
      value: "12345"
    - key: name
      value: some_name
claimMappings:
    - path: org_id
      name: super_cool_claim
    - path: republic
      name: far_away
oidc:
    issuer: sample.issuer.com
    mode: auto
    clientID: CLIENT_ID
    clientSecret: CLIENT_SECRET
---
name: some-name
description: some-description
resources:
    - resource: Access
      access: READ_ACCESS
    - resource: Admin
      access: READ_WRITE_ACCESS
`),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			env, out, errOut := mocks.NewEnvWithConn(nil, t)
			cmd := Command(env)
			cmd.SetOut(out)
			cmd.SetErr(errOut)
			dir := t.TempDir()
			filePath := path.Join(dir, "config")
			require.NoError(t, os.WriteFile(filePath, c.yaml, 0777))
			cmd.SetArgs([]string{"--file=" + filePath})
			assert.NoError(t, cmd.Execute())
			assert.Empty(t, out)
			assert.NotEmpty(t, errOut)
			assert.Equal(t,
				fmt.Sprintf("INFO:\tSuccessfully validated declarative configuration within file %s\n", filePath),
				errOut.String(),
			)
		})
	}
}
