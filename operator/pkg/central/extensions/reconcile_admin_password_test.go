package extensions

import (
	"bytes"
	"testing"

	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/operator/pkg/utils/testutils"
	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestReconcileAdminPassword(t *testing.T) {
	hf := htpasswd.New()
	require.NoError(t, hf.Set(basic.DefaultUsername, "foobar"))
	var buf bytes.Buffer
	require.NoError(t, hf.Write(&buf))

	htpasswdWithSomePassword := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-htpasswd",
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"htpasswd": buf.Bytes(),
		},
	}

	htpasswdWithNoPassword := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-htpasswd",
			Namespace: testutils.TestNamespace,
		},
	}

	plaintextPasswordSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-password",
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"password": []byte("foobarbaz\n"),
		},
	}

	cases := map[string]secretReconciliationTestCase{
		"If no central-htpasswd secret exists and no plaintext secret reference was specified, a password should be automatically generated": {
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-htpasswd": func(t *testing.T, data types.SecretDataMap) {
					plaintextPW := string(data[adminPasswordKey])
					require.NotEmpty(t, plaintextPW)

					htpasswdBytes := data[htpasswdKey]
					hf, err := htpasswd.ReadHashFile(bytes.NewReader(htpasswdBytes))
					require.NoError(t, err)

					assert.True(t, hf.Check(basic.DefaultUsername, plaintextPW))
				},
			},
			VerifyStatus: func(t *testing.T, status *platform.CentralStatus) {
				require.NotNil(t, status.Central)
				require.NotNil(t, status.Central.AdminPassword)
				assert.Contains(t, status.Central.AdminPassword.Info, "A password for the 'admin' user has been automatically generated and stored")
				assert.Contains(t, *status.Central.AdminPassword.SecretReference, "central-htpasswd")
			},
		},
		"If a central-htpasswd secret with a password exists, no password should be generated": {
			Existing: []*v1.Secret{htpasswdWithSomePassword},
			VerifyStatus: func(t *testing.T, status *platform.CentralStatus) {
				require.NotNil(t, status.Central)
				require.NotNil(t, status.Central.AdminPassword)
				assert.Contains(t, status.Central.AdminPassword.Info, "A user-defined central-htpasswd secret was found, containing htpasswd-encoded credentials.")
				assert.Contains(t, *status.Central.AdminPassword.SecretReference, htpasswdWithSomePassword.Name)
			},
		},
		"If a central-htpasswd secret with no password exists, no password should be generated and the user should be informed that basic auth is disabled": {
			Existing: []*v1.Secret{htpasswdWithNoPassword},
			VerifyStatus: func(t *testing.T, status *platform.CentralStatus) {
				require.NotNil(t, status.Central)
				require.NotNil(t, status.Central.AdminPassword)
				assert.Contains(t, status.Central.AdminPassword.Info, "Login with username/password has been disabled")
				assert.Empty(t, status.Central.AdminPassword.SecretReference)
			},
		},
		"If a secret with a plaintext password is referenced, a central-htpasswd secret should be created accordingly": {
			Spec: platform.CentralSpec{
				Central: &platform.CentralComponentSpec{
					AdminPasswordSecret: &platform.LocalSecretReference{
						Name: plaintextPasswordSecret.Name,
					},
				},
			},
			Existing: []*v1.Secret{plaintextPasswordSecret},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-htpasswd": func(t *testing.T, data types.SecretDataMap) {
					htpasswdBytes := data[htpasswdKey]
					hf, err := htpasswd.ReadHashFile(bytes.NewReader(htpasswdBytes))
					require.NoError(t, err)

					assert.True(t, hf.Check(basic.DefaultUsername, "foobarbaz"))
				},
			},
			VerifyStatus: func(t *testing.T, status *platform.CentralStatus) {
				require.NotNil(t, status.Central)
				require.NotNil(t, status.Central.AdminPassword)
				assert.Contains(t, status.Central.AdminPassword.Info, "The admin password is configured to match")
				assert.Contains(t, *status.Central.AdminPassword.SecretReference, "my-password")
			},
		},
		"If a secret is referenced and password generation is disabled create central-htpasswd": {
			Spec: platform.CentralSpec{
				Central: &platform.CentralComponentSpec{
					AdminPasswordSecret: &platform.LocalSecretReference{
						Name: plaintextPasswordSecret.Name,
					},
					AdminPasswordGenerationDisabled: pointer.Bool(true),
				},
			},
			Existing: []*v1.Secret{plaintextPasswordSecret},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-htpasswd": func(t *testing.T, data types.SecretDataMap) {
					require.NotNil(t, data)
				},
			},
			VerifyStatus: func(t *testing.T, status *platform.CentralStatus) {
				require.NotNil(t, status.Central)
				require.NotNil(t, status.Central.AdminPassword)
				assert.Contains(t, status.Central.AdminPassword.Info, "The admin password is configured to match")
				assert.Contains(t, *status.Central.AdminPassword.SecretReference, "my-password")
			},
		},
		"If password generation is disabled no secret should be created": {
			Spec: platform.CentralSpec{
				Central: &platform.CentralComponentSpec{
					AdminPasswordGenerationDisabled: pointer.Bool(true),
				},
			},
			ExpectedNotExistingSecrets: []string{"central-htpasswd"},
			VerifyStatus: func(t *testing.T, status *platform.CentralStatus) {
				require.NotNil(t, status.Central)
				require.NotNil(t, status.Central.AdminPassword)
				assert.Equal(t, status.Central.AdminPassword.Info, "Password generation has been disabled, if you want to enable it set spec.central.adminPasswordGenerationDisabled to false.")
				assert.Empty(t, status.Central.AdminPassword.SecretReference)
			},
		},
	}

	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testSecretReconciliation(t, reconcileAdminPassword, c)
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	secretName := "secret name"
	secretInfo := "some info"

	cases := map[string]struct {
		status       *platform.CentralStatus
		reconcileRun *reconcileAdminPasswordExtensionRun
		shouldReturn bool
	}{
		"should return false if both Info and SecretReference are up-to-date": {
			status: &platform.CentralStatus{
				Central: &platform.CentralComponentStatus{
					AdminPassword: &platform.AdminPasswordStatus{
						Info:            secretInfo,
						SecretReference: &secretName,
					},
				},
			},
			reconcileRun: &reconcileAdminPasswordExtensionRun{
				infoUpdate:         secretInfo,
				passwordSecretName: secretName,
			},
			shouldReturn: false,
		},
		"should return true if Info is not equal to infoUpdate": {
			status: &platform.CentralStatus{
				Central: &platform.CentralComponentStatus{
					AdminPassword: &platform.AdminPasswordStatus{
						Info: "some info",
					},
				},
			},
			reconcileRun: &reconcileAdminPasswordExtensionRun{
				infoUpdate: "other info",
			},
			shouldReturn: true,
		},
		"should return true if SecretReference is not equal to passwordSecretName": {
			status: &platform.CentralStatus{
				Central: &platform.CentralComponentStatus{
					AdminPassword: &platform.AdminPasswordStatus{
						Info:            secretInfo,
						SecretReference: &secretName,
					},
				},
			},
			reconcileRun: &reconcileAdminPasswordExtensionRun{
				infoUpdate:         secretInfo,
				passwordSecretName: "other secret name",
			},
			shouldReturn: true,
		},
		"should return false if status is empty": {
			status:       &platform.CentralStatus{},
			reconcileRun: &reconcileAdminPasswordExtensionRun{},
			shouldReturn: false,
		},
	}

	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := c.reconcileRun.updateStatus(c.status)
			assert.Equal(t, c.shouldReturn, result)
			assert.Equal(t, c.status.Central.AdminPassword.Info, c.reconcileRun.infoUpdate)
			if c.status.Central.AdminPassword.SecretReference != nil {
				assert.Equal(t, *c.status.Central.AdminPassword.SecretReference, c.reconcileRun.passwordSecretName)
			}
		})
	}

}
