package extensions

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/operator/pkg/utils/testutils"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileDBPassword(t *testing.T) {
	const (
		pw1                = "mysecretpassword"
		pw2                = "mysupersecretpassword"
		customPWSecretName = "my-password"
	)
	canonicalPWSecretWithPW1 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canonicalCentralDBPasswordSecretName,
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			centralDBPasswordKey: []byte(pw1),
		},
	}

	canonicalPWSecretWithNoPassword := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canonicalCentralDBPasswordSecretName,
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{},
	}

	customPWSecretWithPW1 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      customPWSecretName,
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"password": []byte(fmt.Sprintf("%s\n", pw1)),
		},
	}

	customPWSecretWithPW2 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      customPWSecretName,
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"password": []byte(fmt.Sprintf("%s\n", pw2)),
		},
	}

	customPWSecretWithInvalidPW := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      customPWSecretName,
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"password": []byte("foo\nbar\n"),
		},
	}

	specWithAutogenPassword := v1alpha1.CentralSpec{
		Central: &v1alpha1.CentralComponentSpec{
			DB: &v1alpha1.CentralDBSpec{},
		},
	}

	specWithUserSpecifiedPassword := v1alpha1.CentralSpec{
		Central: &v1alpha1.CentralComponentSpec{
			DB: &v1alpha1.CentralDBSpec{
				PasswordSecret: &v1alpha1.LocalSecretReference{
					Name: customPWSecretName,
				},
			},
		},
	}

	specWithCanonicalAsUserSpecifiedPassword := v1alpha1.CentralSpec{
		Central: &v1alpha1.CentralComponentSpec{
			DB: &v1alpha1.CentralDBSpec{
				PasswordSecret: &v1alpha1.LocalSecretReference{
					Name: canonicalCentralDBPasswordSecretName,
				},
			},
		},
	}

	cases := map[string]secretReconciliationTestCase{
		"If unmanaged central-db-password secret exists, that secret should be left untouched": {
			Existing: []*v1.Secret{canonicalPWSecretWithPW1},
		},
		"If no central-db-password secret exists and no custom secret reference was specified, a password should be automatically generated": {
			Spec: specWithAutogenPassword,
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				canonicalCentralDBPasswordSecretName: func(t *testing.T, data types.SecretDataMap) {
					_, err := passwordFromSecretData(data)
					assert.NoError(t, err)
				},
			},
		},
		"If a managed central-db-password secret with a password exists, this password should remain unchanged": {
			Spec:            specWithAutogenPassword,
			ExistingManaged: []*v1.Secret{canonicalPWSecretWithPW1},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				canonicalCentralDBPasswordSecretName: func(t *testing.T, data types.SecretDataMap) {
					pw, err := passwordFromSecretData(data)
					require.NoError(t, err)
					assert.Equal(t, pw1, pw)
				},
			},
		},
		"If a managed central-db-password secret with no password exists, a password should be automatically generated": {
			Spec:            specWithAutogenPassword,
			ExistingManaged: []*v1.Secret{canonicalPWSecretWithNoPassword},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				canonicalCentralDBPasswordSecretName: func(t *testing.T, data types.SecretDataMap) {
					_, err := passwordFromSecretData(data)
					assert.NoError(t, err)
				},
			},
		},
		"If an unmanaged central-db-password secret with no password exists, an error should be raised": {
			Spec:          specWithAutogenPassword,
			Existing:      []*v1.Secret{canonicalPWSecretWithNoPassword},
			ExpectedError: "secret must contain a non-empty",
		},
		"If an unmanaged central-db-password secret exists, this password should remain unchanged even without a user-specified password": {
			Spec:     specWithAutogenPassword,
			Existing: []*v1.Secret{canonicalPWSecretWithPW1},
		},
		"If no central-db-password exists, and a user specified password secret was given, the central-db-password secret should be created with this password": {
			Spec:     specWithUserSpecifiedPassword,
			Existing: []*v1.Secret{customPWSecretWithPW1},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				canonicalCentralDBPasswordSecretName: func(t *testing.T, data types.SecretDataMap) {
					pw, err := passwordFromSecretData(data)
					require.NoError(t, err)
					assert.Equal(t, pw1, pw)
				},
			},
		},
		"If a managed central-db-password exists, and a user specified password secret with the same password was given, the central-db-password secret should be left intact": {
			Spec:            specWithUserSpecifiedPassword,
			Existing:        []*v1.Secret{customPWSecretWithPW1},
			ExistingManaged: []*v1.Secret{canonicalPWSecretWithPW1},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				canonicalCentralDBPasswordSecretName: func(t *testing.T, data types.SecretDataMap) {
					pw, err := passwordFromSecretData(data)
					require.NoError(t, err)
					assert.Equal(t, pw1, pw)
				},
			},
		},
		"If a managed central-db-password exists, and a user specified password secret with a different password was given, the central-db-password secret should be updated with this password": {
			Spec:            specWithUserSpecifiedPassword,
			Existing:        []*v1.Secret{customPWSecretWithPW2},
			ExistingManaged: []*v1.Secret{canonicalPWSecretWithPW1},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				canonicalCentralDBPasswordSecretName: func(t *testing.T, data types.SecretDataMap) {
					pw, err := passwordFromSecretData(data)
					require.NoError(t, err)
					assert.Equal(t, pw2, pw)
				},
			},
		},
		"If an unmanaged central-db-password exists, and a user-specified password secret with the same password was given, the central-db-password secret should be left intact": {
			Spec:     specWithUserSpecifiedPassword,
			Existing: []*v1.Secret{customPWSecretWithPW1, canonicalPWSecretWithPW1},
		},
		"If an unmanaged central-db-password exists, and a user-specified password secret with a different password was given, an error should be raised": {
			Spec:          specWithUserSpecifiedPassword,
			Existing:      []*v1.Secret{customPWSecretWithPW2, canonicalPWSecretWithPW1},
			ExpectedError: "existing password does not match expected one",
		},
		"If a user-specified password secret with an invalid password was given, an error should be raised": {
			Spec:          specWithUserSpecifiedPassword,
			Existing:      []*v1.Secret{customPWSecretWithInvalidPW},
			ExpectedError: "secret must contain a non-empty",
		},
		"If the user-specified password secret does not exist, an error should be raised": {
			Spec:          specWithUserSpecifiedPassword,
			ExpectedError: "failed to retrieve central db password secret",
		},
		"If the user-specified password is the canonical one, and that does not exist, an error should be raised": {
			Spec:          specWithCanonicalAsUserSpecifiedPassword,
			ExpectedError: "failed to retrieve central db password secret",
		},
		"If the user-specified password is the canonical one, and that does exist with a valid password, no error should be raised": {
			Spec:     specWithCanonicalAsUserSpecifiedPassword,
			Existing: []*v1.Secret{canonicalPWSecretWithPW1},
		},
		"If the user-specified password is the canonical one, and that does exist with an invalid password, an error should be raised": {
			Spec:          specWithCanonicalAsUserSpecifiedPassword,
			Existing:      []*v1.Secret{canonicalPWSecretWithNoPassword},
			ExpectedError: "secret must contain a non-empty",
		},
		"When using an external DB with specified password secret, that secret should be left untouched": {
			Spec: v1alpha1.CentralSpec{
				Central: &v1alpha1.CentralComponentSpec{
					DB: &v1alpha1.CentralDBSpec{
						ConnectionStringOverride: pointers.String("foo"),
						PasswordSecret: &v1alpha1.LocalSecretReference{
							Name: customPWSecretName,
						},
					},
				},
			},
			Existing: []*v1.Secret{customPWSecretWithPW1, canonicalPWSecretWithPW1},
		},
		"When using an external DB, and no password secret is specified, an error should be raised": {
			Spec: v1alpha1.CentralSpec{
				Central: &v1alpha1.CentralComponentSpec{
					DB: &v1alpha1.CentralDBSpec{
						ConnectionStringOverride: pointers.String("foo"),
					},
				},
			},
			ExpectedError: "setting spec.central.db.passwordSecret is mandatory when using an external DB",
		},
	}

	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testSecretReconciliation(t, reconcileCentralDBPassword, c)
		})
	}
}
