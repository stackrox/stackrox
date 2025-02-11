package extensions

import (
	"context"
	"testing"

	pkgErrors "github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/labels"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type secretReconcilerTestSuite struct {
	suite.Suite

	centralObj *platform.Central
	client     ctrlClient.Client
	ctx        context.Context

	reconciliator *SecretReconciliator
}

func TestSecretReconcilerExtension(t *testing.T) {
	suite.Run(t, new(secretReconcilerTestSuite))
}

func (s *secretReconcilerTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.centralObj = &platform.Central{
		TypeMeta: metav1.TypeMeta{
			APIVersion: platform.CentralGVK.GroupVersion().String(),
			Kind:       platform.CentralGVK.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stackrox-central-services",
			Namespace: testutils.TestNamespace,
			UID:       k8sTypes.UID(uuid.NewV4().String()),
		},
	}

	existingSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"secret-name": []byte("existing-secret"),
			"managed":     []byte("false"),
		},
	}

	existingOwnedSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-managed-secret",
			Namespace: testutils.TestNamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(s.centralObj, platform.CentralGVK),
			},
			Labels: labels.DefaultLabels(),
		},
		Data: map[string][]byte{
			"secret-name": []byte("existing-managed-secret"),
			"managed":     []byte("true"),
		},
	}

	s.client = fake.NewClientBuilder().WithObjects(existingSecret, existingOwnedSecret).Build()

	s.reconciliator = NewSecretReconciliator(s.client, s.client, s.centralObj)
}

func (s *secretReconcilerTestSuite) Test_ShouldNotExist_OnNonExisting_ShouldDoNothing() {
	err := s.reconciliator.DeleteSecret(s.ctx, "absent-secret")
	s.Require().NoError(err)

	dummy := &v1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "absent-secret"}
	err = s.client.Get(context.Background(), key, dummy)
	s.True(errors.IsNotFound(err))
}

func (s *secretReconcilerTestSuite) Test_ShouldNotExist_OnExistingManaged_ShouldDelete() {
	err := s.reconciliator.DeleteSecret(s.ctx, "existing-managed-secret")
	s.Require().NoError(err)

	dummy := &v1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "existing-managed-secret"}
	err = s.client.Get(context.Background(), key, dummy)
	s.True(errors.IsNotFound(err))
}

func (s *secretReconcilerTestSuite) Test_ShouldNotExist_OnExistingUnmanaged_ShouldDoNothing() {
	err := s.reconciliator.DeleteSecret(s.ctx, "existing-secret")
	s.Require().NoError(err)

	dummy := &v1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "existing-managed-secret"}
	err = s.client.Get(context.Background(), key, dummy)
	s.NoError(err)
}

func (s *secretReconcilerTestSuite) Test_ShouldExist_OnNonExisting_ShouldCreateSecretWithOwnerRef_Success() {
	validateFn := func(types.SecretDataMap, bool) error {
		s.Require().Fail("this function should not be called")
		panic("unexpected")
	}
	// this ensures that we check for the existence of a unique created secret
	var markerID string
	generateFn := func(_ types.SecretDataMap) (types.SecretDataMap, error) {
		markerID = uuid.NewV4().String()
		return types.SecretDataMap{
			"generated": []byte(markerID),
		}, nil
	}

	err := s.reconciliator.EnsureSecret(s.ctx, "absent-secret", validateFn, generateFn, nil)
	s.Require().NoError(err)
	s.NotEmpty(markerID, "generate function has not been called")

	secret := &v1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "absent-secret"}
	err = s.client.Get(context.Background(), key, secret)
	s.Require().NoError(err)

	s.EqualValues(secret.GetOwnerReferences(), []metav1.OwnerReference{*metav1.NewControllerRef(s.centralObj, platform.CentralGVK)})

	s.Equal(markerID, string(secret.Data["generated"]))
}

func (s *secretReconcilerTestSuite) Test_ShouldExist_OnNonExisting_ShouldCreateSecretWithOwnerRef_Failure() {
	validateFn := func(types.SecretDataMap, bool) error {
		s.Require().Fail("this function should not be called")
		panic("unexpected")
	}
	failGenerationErr := pkgErrors.New("generation failed")
	generateFn := func(_ types.SecretDataMap) (types.SecretDataMap, error) {
		return nil, failGenerationErr
	}

	err := s.reconciliator.EnsureSecret(s.ctx, "absent-secret", validateFn, generateFn, nil)
	s.ErrorIs(err, failGenerationErr)

	secret := &v1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "absent-secret"}
	err = s.client.Get(context.Background(), key, secret)

	s.Truef(errors.IsNotFound(err), "secret should still be missing, found %+v", secret)
}

func (s *secretReconcilerTestSuite) Test_ShouldExist_OnExistingManaged_PassingValidation_ShouldDoNothing() {
	initSecret := &v1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "existing-managed-secret"}
	err := s.client.Get(context.Background(), key, initSecret)
	s.Require().NoError(err)

	validated := false
	validateFn := func(data types.SecretDataMap, managed bool) error {
		s.Equal("existing-managed-secret", string(data["secret-name"]))
		s.True(managed)
		validated = true
		return nil
	}

	generateFn := func(_ types.SecretDataMap) (types.SecretDataMap, error) {
		s.Require().Fail("this function should not be called")
		panic("unexpected")
	}

	err = s.reconciliator.EnsureSecret(s.ctx, "existing-managed-secret", validateFn, generateFn, nil)
	s.Require().NoError(err)
	s.True(validated)

	secret := &v1.Secret{}
	err = s.client.Get(context.Background(), key, secret)
	s.Require().NoError(err)

	assert.Equal(s.T(), initSecret, secret)
}

func (s *secretReconcilerTestSuite) Test_ShouldExist_OnExistingManaged_FailingValidation_ShouldFix() {
	failValidationErr := pkgErrors.New("failed validation")
	validateFn := func(data types.SecretDataMap, managed bool) error {
		s.Equal("existing-managed-secret", string(data["secret-name"]))
		s.True(managed)
		return failValidationErr
	}

	generateFn := func(_ types.SecretDataMap) (types.SecretDataMap, error) {
		return types.SecretDataMap{
			"new-secret-data": []byte("foo"),
		}, nil
	}

	err := s.reconciliator.EnsureSecret(s.ctx, "existing-managed-secret", validateFn, generateFn, nil)
	s.NoError(err)

	secret := &v1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "existing-managed-secret"}
	err = s.client.Get(context.Background(), key, secret)
	s.Require().NoError(err)

	s.Equal("foo", string(secret.Data["new-secret-data"]))
}

func (s *secretReconcilerTestSuite) Test_ShouldExist_OnExistingUnmanaged_PassingValidation_ShouldDoNothing() {
	initSecret := &v1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "existing-secret"}
	err := s.client.Get(context.Background(), key, initSecret)
	s.Require().NoError(err)

	validated := false
	validateFn := func(data types.SecretDataMap, managed bool) error {
		s.Equal("existing-secret", string(data["secret-name"]))
		s.False(managed)
		validated = true
		return nil
	}

	generateFn := func(_ types.SecretDataMap) (types.SecretDataMap, error) {
		s.Require().Fail("this function should not be called")
		panic("unexpected")
	}

	err = s.reconciliator.EnsureSecret(s.ctx, "existing-secret", validateFn, generateFn, nil)
	s.Require().NoError(err)
	s.True(validated)

	secret := &v1.Secret{}
	err = s.client.Get(context.Background(), key, secret)
	s.Require().NoError(err)
	assert.Equal(s.T(), initSecret, secret)
}

func (s *secretReconcilerTestSuite) Test_ShouldExist_OnExistingUnmanaged_FailingValidation_ShouldDoNothingAndFail() {
	initSecret := &v1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "existing-secret"}
	err := s.client.Get(context.Background(), key, initSecret)
	s.Require().NoError(err)

	failValidationErr := pkgErrors.New("failed validation")
	validateFn := func(data types.SecretDataMap, managed bool) error {
		s.Equal("existing-secret", string(data["secret-name"]))
		s.False(managed)
		return failValidationErr
	}

	generateFn := func(_ types.SecretDataMap) (types.SecretDataMap, error) {
		s.Require().Fail("this function should not be called")
		panic("unexpected")
	}

	err = s.reconciliator.EnsureSecret(s.ctx, "existing-secret", validateFn, generateFn, nil)
	s.ErrorIs(err, failValidationErr)

	secret := &v1.Secret{}
	err = s.client.Get(context.Background(), key, secret)
	s.Require().NoError(err)

	assert.Equal(s.T(), initSecret, secret)
}
