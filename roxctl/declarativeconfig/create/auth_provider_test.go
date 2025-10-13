package create

import (
	"io"
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAuthProvider_Failures(t *testing.T) {

	cases := map[string]struct {
		args   []string
		errOut string
		err    error
	}{
		"missing name flag": {
			args: []string{
				"auth-provider",
				"openshift-auth",
				"--ui-endpoint=localhost:8000",
			},
			errOut: `Error: if any flags in the group [name ui-endpoint] are set they must all be set; missing [name]
`,
		},
		"missing ui-endpoint flag": {
			args: []string{
				"auth-provider",
				"openshift-auth",
				"--name=some-name",
			},
			errOut: `Error: if any flags in the group [name ui-endpoint] are set they must all be set; missing [ui-endpoint]
`,
		},
		"invalid number of groups keys": {
			args: []string{
				"auth-provider",
				"openshift-auth",
				"--name=some-name",
				"--ui-endpoint=localhost:8000",
				"--groups-key=email",
				"--groups-value=example@example.com",
				"--groups-role=Admin",
				"--groups-key=another-one",
			},
			err: errox.InvalidArgs,
		},
		"invalid number of groups values": {
			args: []string{
				"auth-provider",
				"openshift-auth",
				"--name=some-name",
				"--ui-endpoint=localhost:8000",
				"--groups-key=email",
				"--groups-value=example@example.com",
				"--groups-role=Admin",
				"--groups-value=another-one",
			},
			err: errox.InvalidArgs,
		},
		"invalid number of groups roles": {
			args: []string{
				"auth-provider",
				"openshift-auth",
				"--name=some-name",
				"--ui-endpoint=localhost:8000",
				"--groups-key=email",
				"--groups-value=example@example.com",
				"--groups-role=Admin",
				"--groups-role=another-one",
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

func TestCreateAuthProvider_SAML_Failure(t *testing.T) {
	env, _, _ := mocks.NewEnvWithConn(nil, t)
	cmd := Command(env)

	args := []string{
		"auth-provider",
		"saml",
		"--sp-issuer=something",
		"--idp-cert=non-existent/file/path",
		"--sso-url=something",
		"--idp-issuer=something",
	}
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	assert.ErrorIs(t, err, errox.NotFound)
}

func TestCreateAuthProvider_UserPKI_Failure(t *testing.T) {
	env, _, _ := mocks.NewEnvWithConn(nil, t)
	cmd := Command(env)

	args := []string{
		"auth-provider",
		"userpki",
		"--ca-file=non-existent/file/path",
	}
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	assert.ErrorIs(t, err, errox.NotFound)
}

func TestCreateAuthProvider_OIDC_Success(t *testing.T) {
	args := []string{
		"auth-provider",
		"oidc",
		"--name=some-name",
		"--ui-endpoint=localhost:8000",
		"--groups-key=email",
		"--groups-value=example@example.com",
		"--groups-role=Admin",
		"--groups-key=userid",
		"--groups-value=someid",
		"--groups-role=Analyst",
		"--minimum-access-role=Analyst",
		"--extra-ui-endpoints=localhost:9090",
		"--extra-ui-endpoints=localhost:10010",
		"--required-attributes=org_id=12345,name=some_name",
		"--issuer=sample.issuer.com",
		"--mode=auto",
		"--client-id=CLIENT_ID",
		"--client-secret=CLIENT_SECRET",
		"--claim-mappings=org_id=super_cool_claim,republic=far_away",
	}

	expectedYAML := `name: some-name
minimumRole: Analyst
uiEndpoint: localhost:8000
extraUIEndpoints:
    - localhost:9090
    - localhost:10010
groups:
    - key: email
      value: example@example.com
      role: Admin
    - key: userid
      value: someid
      role: Analyst
requiredAttributes:
    - key: name
      value: some_name
    - key: org_id
      value: "12345"
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
`

	runSuccessfulCommandTest(t, args, expectedYAML)
}

func TestCreateAuthProvider_SAML_Success(t *testing.T) {
	dir := t.TempDir()

	filePath := path.Join(dir, "idp-cert")
	f, err := os.Create(filePath)
	assert.NoError(t, err)
	defer utils.IgnoreError(f.Close)
	_, err = f.Write([]byte(`-----BEGIN CERTIFICATE-----
MIIECTCCA3KgAwIBAgIUDnU7Oa0fU9GFOwU7EWJP3HsRchEwDQYJKoZIhvcNAQEL
BQAwgZkxCzAJBgNVBAYTAlVTMRAwDgYDVQQIDAdNb250YW5hMRAwDgYDVQQHDAdC
b3plbWFuMREwDwYDVQQKDAhTYXd0b290aDEYMBYGA1UECwwPQ29uc3VsdGluZ18x
MDI0MRgwFgYDVQQDDA93d3cud29sZnNzbC5jb20xHzAdBgkqhkiG9w0BCQEWEGlu
Zm9Ad29sZnNzbC5jb20wHhcNMjIxMjE2MjExNzQ5WhcNMjUwOTExMjExNzQ5WjCB
mTELMAkGA1UEBhMCVVMxEDAOBgNVBAgMB01vbnRhbmExEDAOBgNVBAcMB0JvemVt
YW4xETAPBgNVBAoMCFNhd3Rvb3RoMRgwFgYDVQQLDA9Db25zdWx0aW5nXzEwMjQx
GDAWBgNVBAMMD3d3dy53b2xmc3NsLmNvbTEfMB0GCSqGSIb3DQEJARYQaW5mb0B3
b2xmc3NsLmNvbTCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAzazdR+y+tyTD
YxtUmHnhxzEWWdadd52N4ovtBBeyxuvkm5G+MVBil1i1fynes3EkC7+XCX8m3C3s
qC6yZCt6KzUZLaKAy5n9lHEbI41U2y5ijYEILfQkcids+cmO20x1upsB+D8Y9OZ/
+1eUksyIxLQAwqrU5YgYsxEvc8DWKQkCAwEAAaOCAUowggFGMB0GA1UdDgQWBBTT
Io8oLOAF7tPtw3E9ybI2Oh2/qDCB2QYDVR0jBIHRMIHOgBTTIo8oLOAF7tPtw3E9
ybI2Oh2/qKGBn6SBnDCBmTELMAkGA1UEBhMCVVMxEDAOBgNVBAgMB01vbnRhbmEx
EDAOBgNVBAcMB0JvemVtYW4xETAPBgNVBAoMCFNhd3Rvb3RoMRgwFgYDVQQLDA9D
b25zdWx0aW5nXzEwMjQxGDAWBgNVBAMMD3d3dy53b2xmc3NsLmNvbTEfMB0GCSqG
SIb3DQEJARYQaW5mb0B3b2xmc3NsLmNvbYIUDnU7Oa0fU9GFOwU7EWJP3HsRchEw
DAYDVR0TBAUwAwEB/zAcBgNVHREEFTATggtleGFtcGxlLmNvbYcEfwAAATAdBgNV
HSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDQYJKoZIhvcNAQELBQADgYEAuIC/
svWDlVGBan5BhynXw8nGm2DkZaEElx0bO+kn+kPWiWo8nr8o0XU3IfMNZBeyoy2D
Uv9X8EKpSKrYhOoNgAVxCqojtGzG1n8TSvSCueKBrkaMWfvDjG1b8zLshvBu2ip4
q/I2+0j6dAkOGcK/68z7qQXByeGri3n28a1Kn6o=
-----END CERTIFICATE-----
`))
	assert.NoError(t, err)

	args := []string{
		"auth-provider",
		"saml",
		"--name=some-name",
		"--ui-endpoint=localhost:8000",
		"--groups-key=email",
		"--groups-value=example@example.com",
		"--groups-role=Admin",
		"--groups-key=userid",
		"--groups-value=someid",
		"--groups-role=Analyst",
		"--minimum-access-role=Analyst",
		"--extra-ui-endpoints=localhost:9090",
		"--extra-ui-endpoints=localhost:10010",
		"--required-attributes=org_id=12345,name=some_name",
		"--sp-issuer=some-random-issuer",
		"--idp-cert=" + filePath,
		"--sso-url=some-sso.url",
		"--name-id-format=some-format",
		"--idp-issuer=my.cool.issuer",
	}

	expectedYAML := `name: some-name
minimumRole: Analyst
uiEndpoint: localhost:8000
extraUIEndpoints:
    - localhost:9090
    - localhost:10010
groups:
    - key: email
      value: example@example.com
      role: Admin
    - key: userid
      value: someid
      role: Analyst
requiredAttributes:
    - key: name
      value: some_name
    - key: org_id
      value: "12345"
saml:
    spIssuer: some-random-issuer
    cert: |
        -----BEGIN CERTIFICATE-----
        MIIECTCCA3KgAwIBAgIUDnU7Oa0fU9GFOwU7EWJP3HsRchEwDQYJKoZIhvcNAQEL
        BQAwgZkxCzAJBgNVBAYTAlVTMRAwDgYDVQQIDAdNb250YW5hMRAwDgYDVQQHDAdC
        b3plbWFuMREwDwYDVQQKDAhTYXd0b290aDEYMBYGA1UECwwPQ29uc3VsdGluZ18x
        MDI0MRgwFgYDVQQDDA93d3cud29sZnNzbC5jb20xHzAdBgkqhkiG9w0BCQEWEGlu
        Zm9Ad29sZnNzbC5jb20wHhcNMjIxMjE2MjExNzQ5WhcNMjUwOTExMjExNzQ5WjCB
        mTELMAkGA1UEBhMCVVMxEDAOBgNVBAgMB01vbnRhbmExEDAOBgNVBAcMB0JvemVt
        YW4xETAPBgNVBAoMCFNhd3Rvb3RoMRgwFgYDVQQLDA9Db25zdWx0aW5nXzEwMjQx
        GDAWBgNVBAMMD3d3dy53b2xmc3NsLmNvbTEfMB0GCSqGSIb3DQEJARYQaW5mb0B3
        b2xmc3NsLmNvbTCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAzazdR+y+tyTD
        YxtUmHnhxzEWWdadd52N4ovtBBeyxuvkm5G+MVBil1i1fynes3EkC7+XCX8m3C3s
        qC6yZCt6KzUZLaKAy5n9lHEbI41U2y5ijYEILfQkcids+cmO20x1upsB+D8Y9OZ/
        +1eUksyIxLQAwqrU5YgYsxEvc8DWKQkCAwEAAaOCAUowggFGMB0GA1UdDgQWBBTT
        Io8oLOAF7tPtw3E9ybI2Oh2/qDCB2QYDVR0jBIHRMIHOgBTTIo8oLOAF7tPtw3E9
        ybI2Oh2/qKGBn6SBnDCBmTELMAkGA1UEBhMCVVMxEDAOBgNVBAgMB01vbnRhbmEx
        EDAOBgNVBAcMB0JvemVtYW4xETAPBgNVBAoMCFNhd3Rvb3RoMRgwFgYDVQQLDA9D
        b25zdWx0aW5nXzEwMjQxGDAWBgNVBAMMD3d3dy53b2xmc3NsLmNvbTEfMB0GCSqG
        SIb3DQEJARYQaW5mb0B3b2xmc3NsLmNvbYIUDnU7Oa0fU9GFOwU7EWJP3HsRchEw
        DAYDVR0TBAUwAwEB/zAcBgNVHREEFTATggtleGFtcGxlLmNvbYcEfwAAATAdBgNV
        HSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDQYJKoZIhvcNAQELBQADgYEAuIC/
        svWDlVGBan5BhynXw8nGm2DkZaEElx0bO+kn+kPWiWo8nr8o0XU3IfMNZBeyoy2D
        Uv9X8EKpSKrYhOoNgAVxCqojtGzG1n8TSvSCueKBrkaMWfvDjG1b8zLshvBu2ip4
        q/I2+0j6dAkOGcK/68z7qQXByeGri3n28a1Kn6o=
        -----END CERTIFICATE-----
    ssoURL: some-sso.url
    nameIdFormat: some-format
    idpIssuer: my.cool.issuer
`

	runSuccessfulCommandTest(t, args, expectedYAML)
}

func TestCreateAuthProvider_UserPKI_Success(t *testing.T) {
	dir := t.TempDir()

	filePath := path.Join(dir, "ca-file")
	f, err := os.Create(filePath)
	assert.NoError(t, err)
	defer utils.IgnoreError(f.Close)
	_, err = f.Write([]byte(`-----BEGIN CERTIFICATE-----
MIIECTCCA3KgAwIBAgIUDnU7Oa0fU9GFOwU7EWJP3HsRchEwDQYJKoZIhvcNAQEL
BQAwgZkxCzAJBgNVBAYTAlVTMRAwDgYDVQQIDAdNb250YW5hMRAwDgYDVQQHDAdC
b3plbWFuMREwDwYDVQQKDAhTYXd0b290aDEYMBYGA1UECwwPQ29uc3VsdGluZ18x
MDI0MRgwFgYDVQQDDA93d3cud29sZnNzbC5jb20xHzAdBgkqhkiG9w0BCQEWEGlu
Zm9Ad29sZnNzbC5jb20wHhcNMjIxMjE2MjExNzQ5WhcNMjUwOTExMjExNzQ5WjCB
mTELMAkGA1UEBhMCVVMxEDAOBgNVBAgMB01vbnRhbmExEDAOBgNVBAcMB0JvemVt
YW4xETAPBgNVBAoMCFNhd3Rvb3RoMRgwFgYDVQQLDA9Db25zdWx0aW5nXzEwMjQx
GDAWBgNVBAMMD3d3dy53b2xmc3NsLmNvbTEfMB0GCSqGSIb3DQEJARYQaW5mb0B3
b2xmc3NsLmNvbTCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAzazdR+y+tyTD
YxtUmHnhxzEWWdadd52N4ovtBBeyxuvkm5G+MVBil1i1fynes3EkC7+XCX8m3C3s
qC6yZCt6KzUZLaKAy5n9lHEbI41U2y5ijYEILfQkcids+cmO20x1upsB+D8Y9OZ/
+1eUksyIxLQAwqrU5YgYsxEvc8DWKQkCAwEAAaOCAUowggFGMB0GA1UdDgQWBBTT
Io8oLOAF7tPtw3E9ybI2Oh2/qDCB2QYDVR0jBIHRMIHOgBTTIo8oLOAF7tPtw3E9
ybI2Oh2/qKGBn6SBnDCBmTELMAkGA1UEBhMCVVMxEDAOBgNVBAgMB01vbnRhbmEx
EDAOBgNVBAcMB0JvemVtYW4xETAPBgNVBAoMCFNhd3Rvb3RoMRgwFgYDVQQLDA9D
b25zdWx0aW5nXzEwMjQxGDAWBgNVBAMMD3d3dy53b2xmc3NsLmNvbTEfMB0GCSqG
SIb3DQEJARYQaW5mb0B3b2xmc3NsLmNvbYIUDnU7Oa0fU9GFOwU7EWJP3HsRchEw
DAYDVR0TBAUwAwEB/zAcBgNVHREEFTATggtleGFtcGxlLmNvbYcEfwAAATAdBgNV
HSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDQYJKoZIhvcNAQELBQADgYEAuIC/
svWDlVGBan5BhynXw8nGm2DkZaEElx0bO+kn+kPWiWo8nr8o0XU3IfMNZBeyoy2D
Uv9X8EKpSKrYhOoNgAVxCqojtGzG1n8TSvSCueKBrkaMWfvDjG1b8zLshvBu2ip4
q/I2+0j6dAkOGcK/68z7qQXByeGri3n28a1Kn6o=
-----END CERTIFICATE-----
`))
	assert.NoError(t, err)

	args := []string{
		"auth-provider",
		"userpki",
		"--name=some-name",
		"--ui-endpoint=localhost:8000",
		"--groups-key=email",
		"--groups-value=example@example.com",
		"--groups-role=Admin",
		"--groups-key=userid",
		"--groups-value=someid",
		"--groups-role=Analyst",
		"--minimum-access-role=Analyst",
		"--extra-ui-endpoints=localhost:9090",
		"--extra-ui-endpoints=localhost:10010",
		"--required-attributes=org_id=12345,name=some_name",
		"--ca-file=" + filePath,
	}

	expectedYAML := `name: some-name
minimumRole: Analyst
uiEndpoint: localhost:8000
extraUIEndpoints:
    - localhost:9090
    - localhost:10010
groups:
    - key: email
      value: example@example.com
      role: Admin
    - key: userid
      value: someid
      role: Analyst
requiredAttributes:
    - key: name
      value: some_name
    - key: org_id
      value: "12345"
userpki:
    certificateAuthorities: |
        -----BEGIN CERTIFICATE-----
        MIIECTCCA3KgAwIBAgIUDnU7Oa0fU9GFOwU7EWJP3HsRchEwDQYJKoZIhvcNAQEL
        BQAwgZkxCzAJBgNVBAYTAlVTMRAwDgYDVQQIDAdNb250YW5hMRAwDgYDVQQHDAdC
        b3plbWFuMREwDwYDVQQKDAhTYXd0b290aDEYMBYGA1UECwwPQ29uc3VsdGluZ18x
        MDI0MRgwFgYDVQQDDA93d3cud29sZnNzbC5jb20xHzAdBgkqhkiG9w0BCQEWEGlu
        Zm9Ad29sZnNzbC5jb20wHhcNMjIxMjE2MjExNzQ5WhcNMjUwOTExMjExNzQ5WjCB
        mTELMAkGA1UEBhMCVVMxEDAOBgNVBAgMB01vbnRhbmExEDAOBgNVBAcMB0JvemVt
        YW4xETAPBgNVBAoMCFNhd3Rvb3RoMRgwFgYDVQQLDA9Db25zdWx0aW5nXzEwMjQx
        GDAWBgNVBAMMD3d3dy53b2xmc3NsLmNvbTEfMB0GCSqGSIb3DQEJARYQaW5mb0B3
        b2xmc3NsLmNvbTCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAzazdR+y+tyTD
        YxtUmHnhxzEWWdadd52N4ovtBBeyxuvkm5G+MVBil1i1fynes3EkC7+XCX8m3C3s
        qC6yZCt6KzUZLaKAy5n9lHEbI41U2y5ijYEILfQkcids+cmO20x1upsB+D8Y9OZ/
        +1eUksyIxLQAwqrU5YgYsxEvc8DWKQkCAwEAAaOCAUowggFGMB0GA1UdDgQWBBTT
        Io8oLOAF7tPtw3E9ybI2Oh2/qDCB2QYDVR0jBIHRMIHOgBTTIo8oLOAF7tPtw3E9
        ybI2Oh2/qKGBn6SBnDCBmTELMAkGA1UEBhMCVVMxEDAOBgNVBAgMB01vbnRhbmEx
        EDAOBgNVBAcMB0JvemVtYW4xETAPBgNVBAoMCFNhd3Rvb3RoMRgwFgYDVQQLDA9D
        b25zdWx0aW5nXzEwMjQxGDAWBgNVBAMMD3d3dy53b2xmc3NsLmNvbTEfMB0GCSqG
        SIb3DQEJARYQaW5mb0B3b2xmc3NsLmNvbYIUDnU7Oa0fU9GFOwU7EWJP3HsRchEw
        DAYDVR0TBAUwAwEB/zAcBgNVHREEFTATggtleGFtcGxlLmNvbYcEfwAAATAdBgNV
        HSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDQYJKoZIhvcNAQELBQADgYEAuIC/
        svWDlVGBan5BhynXw8nGm2DkZaEElx0bO+kn+kPWiWo8nr8o0XU3IfMNZBeyoy2D
        Uv9X8EKpSKrYhOoNgAVxCqojtGzG1n8TSvSCueKBrkaMWfvDjG1b8zLshvBu2ip4
        q/I2+0j6dAkOGcK/68z7qQXByeGri3n28a1Kn6o=
        -----END CERTIFICATE-----
`

	runSuccessfulCommandTest(t, args, expectedYAML)
}

func TestCreateAuthProvider_OpenShiftAuth_Success(t *testing.T) {
	args := []string{
		"auth-provider",
		"openshift-auth",
		"--name=some-name",
		"--ui-endpoint=localhost:8000",
		"--groups-key=email",
		"--groups-value=example@example.com",
		"--groups-role=Admin",
		"--groups-key=userid",
		"--groups-value=someid",
		"--groups-role=Analyst",
		"--minimum-access-role=Analyst",
		"--extra-ui-endpoints=localhost:9090",
		"--extra-ui-endpoints=localhost:10010",
		"--required-attributes=org_id=12345,name=some_name",
	}

	expectedYAML := `name: some-name
minimumRole: Analyst
uiEndpoint: localhost:8000
extraUIEndpoints:
    - localhost:9090
    - localhost:10010
groups:
    - key: email
      value: example@example.com
      role: Admin
    - key: userid
      value: someid
      role: Analyst
requiredAttributes:
    - key: name
      value: some_name
    - key: org_id
      value: "12345"
openshift:
    enable: true
`

	runSuccessfulCommandTest(t, args, expectedYAML)
}

func TestCreateAuthProvider_IAP_Success(t *testing.T) {
	args := []string{
		"auth-provider",
		"iap",
		"--name=some-name",
		"--ui-endpoint=localhost:8000",
		"--groups-key=email",
		"--groups-value=example@example.com",
		"--groups-role=Admin",
		"--groups-key=userid",
		"--groups-value=someid",
		"--groups-role=Analyst",
		"--minimum-access-role=Analyst",
		"--extra-ui-endpoints=localhost:9090",
		"--extra-ui-endpoints=localhost:10010",
		"--required-attributes=org_id=12345,name=some_name",
		"--audience=some-audience",
	}

	expectedYAML := `name: some-name
minimumRole: Analyst
uiEndpoint: localhost:8000
extraUIEndpoints:
    - localhost:9090
    - localhost:10010
groups:
    - key: email
      value: example@example.com
      role: Admin
    - key: userid
      value: someid
      role: Analyst
requiredAttributes:
    - key: name
      value: some_name
    - key: org_id
      value: "12345"
iap:
    audience: some-audience
`

	runSuccessfulCommandTest(t, args, expectedYAML)
}

func runSuccessfulCommandTest(t *testing.T, args []string, expectedYAML string) {
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

func TestAuthProvider_WriteToK8sObject(t *testing.T) {
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

			authProviderCmd := authProviderCmd{}
			err := authProviderCmd.Construct(cmd)
			require.NoError(t, err)
			assert.Equal(t, c.shouldWriteToK8sObject, authProviderCmd.configMap != "" || authProviderCmd.secret != "")
		})
	}
}
