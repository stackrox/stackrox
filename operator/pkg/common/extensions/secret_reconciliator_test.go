package extensions

import (
	"context"
	"strings"
	"testing"
	"text/tabwriter"

	pkgErrors "github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/common/labels"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/operator/pkg/utils/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	s.centralObj = newCentral()

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

	s.reconciliator = NewSecretReconciliator(s.client, s.client, s.centralObj, OwnershipStrategyOwnerReference)
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

	err := s.reconciliator.EnsureSecret(s.ctx, "absent-secret", validateFn, generateFn)
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

	err := s.reconciliator.EnsureSecret(s.ctx, "absent-secret", validateFn, generateFn)
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

	err = s.reconciliator.EnsureSecret(s.ctx, "existing-managed-secret", validateFn, generateFn)
	s.Require().NoError(err)
	s.True(validated)

	secret := &v1.Secret{}
	err = s.client.Get(context.Background(), key, secret)
	s.Require().NoError(err)

	s.Equal(initSecret, secret)
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

	err := s.reconciliator.EnsureSecret(s.ctx, "existing-managed-secret", validateFn, generateFn)
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

	err = s.reconciliator.EnsureSecret(s.ctx, "existing-secret", validateFn, generateFn)
	s.Require().NoError(err)
	s.True(validated)

	secret := &v1.Secret{}
	err = s.client.Get(context.Background(), key, secret)
	s.Require().NoError(err)
	s.Equal(initSecret, secret)
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

	err = s.reconciliator.EnsureSecret(s.ctx, "existing-secret", validateFn, generateFn)
	s.ErrorIs(err, failValidationErr)

	secret := &v1.Secret{}
	err = s.client.Get(context.Background(), key, secret)
	s.Require().NoError(err)

	s.Equal(initSecret, secret)
}

// TestSecretReconciler_EnsureSecret_Ownership tests the ownership strategy logic of the secret reconciler
func TestSecretReconciler_EnsureSecret_Ownership(t *testing.T) {

	// Test matrix shown below with expected results. Easier to read than the test table
	// (the test will fail if the matrix is not in sync with the test table)
	const expectedMatrix = `
Strategy           State                        ExpectOwnerRef    ExpectLabel
owner-reference    secretDoesNotExist           x                 x
owner-reference    unmanaged
owner-reference    hasOwnerRef                  x                 x
owner-reference    hasLabel                     x                 x
owner-reference    hasWrongLabel
owner-reference    hasOwnerRef,hasLabel         x                 x
owner-reference    hasOwnerRef,hasWrongLabel    x                 x
label              secretDoesNotExist                             x
label              unmanaged
label              hasOwnerRef                                    x
label              hasLabel                                       x
label              hasWrongLabel
label              hasOwnerRef,hasLabel                           x
label              hasOwnerRef,hasWrongLabel                      x

`

	tests := []struct {
		strategy         OwnershipStrategy
		state            string
		expectedOwnerRef bool
		expectedLabel    bool
	}{
		{
			strategy:         OwnershipStrategyOwnerReference,
			state:            "secretDoesNotExist",
			expectedOwnerRef: true,
			expectedLabel:    true,
		},
		{
			strategy:         OwnershipStrategyOwnerReference,
			state:            "unmanaged",
			expectedOwnerRef: false,
			expectedLabel:    false,
		},
		{
			strategy:         OwnershipStrategyOwnerReference,
			state:            "hasOwnerRef",
			expectedOwnerRef: true,
			expectedLabel:    true,
		},
		{
			strategy:         OwnershipStrategyOwnerReference,
			state:            "hasLabel",
			expectedOwnerRef: true,
			expectedLabel:    true,
		},
		{
			strategy:         OwnershipStrategyOwnerReference,
			state:            "hasWrongLabel",
			expectedOwnerRef: false,
			expectedLabel:    false,
		},
		{
			strategy:         OwnershipStrategyOwnerReference,
			state:            "hasOwnerRef,hasLabel",
			expectedOwnerRef: true,
			expectedLabel:    true,
		},
		{
			strategy:         OwnershipStrategyOwnerReference,
			state:            "hasOwnerRef,hasWrongLabel",
			expectedOwnerRef: true,
			expectedLabel:    true,
		},
		{
			strategy:         OwnershipStrategyLabel,
			state:            "secretDoesNotExist",
			expectedOwnerRef: false,
			expectedLabel:    true,
		},
		{
			strategy:         OwnershipStrategyLabel,
			state:            "unmanaged",
			expectedOwnerRef: false,
			expectedLabel:    false,
		},
		{
			strategy:         OwnershipStrategyLabel,
			state:            "hasOwnerRef",
			expectedOwnerRef: false,
			expectedLabel:    true,
		},
		{
			strategy:         OwnershipStrategyLabel,
			state:            "hasLabel",
			expectedOwnerRef: false,
			expectedLabel:    true,
		},
		{
			strategy:         OwnershipStrategyLabel,
			state:            "hasWrongLabel",
			expectedOwnerRef: false,
			expectedLabel:    false,
		},
		{
			strategy:         OwnershipStrategyLabel,
			state:            "hasOwnerRef,hasLabel",
			expectedOwnerRef: false,
			expectedLabel:    true,
		},
		{
			strategy:         OwnershipStrategyLabel,
			state:            "hasOwnerRef,hasWrongLabel",
			expectedOwnerRef: false,
			expectedLabel:    true, // special case. Assumes presence of ownerRef is a stronger signal than label to determine ownership
		},
	}

	{
		// Validate test matrix

		wr := &strings.Builder{}
		wr.WriteString("\n")

		tbl := tabwriter.NewWriter(wr, 0, 0, 4, ' ', tabwriter.DiscardEmptyColumns)
		_, err := tbl.Write([]byte("Strategy\tState\tExpectOwnerRef\tExpectLabel\n"))
		require.NoError(t, err)

		for _, test := range tests {
			var ts = string(test.strategy) + "\t" + test.state + "\t"
			if test.expectedOwnerRef {
				ts += "x"
			}
			ts += "\t"
			if test.expectedLabel {
				ts += "x"
			}
			ts += "\n"
			_, err := tbl.Write([]byte(ts))
			require.NoError(t, err)
		}
		require.NoError(t, tbl.Flush())

		gotMatrix := wr.String()
		// remove trailing whitespace
		clean := ""
		for _, line := range strings.Split(gotMatrix, "\n") {
			clean += strings.TrimSpace(line) + "\n"
		}
		gotMatrix = clean

		require.Equal(t, expectedMatrix, gotMatrix, "test matrix is not in sync with the test table")
	}

	// dummy test dependencies
	var (
		secretName string
		central    *platform.Central
		generate   func(dataMap types.SecretDataMap) (types.SecretDataMap, error)
		validate   func(dataMap types.SecretDataMap, b bool) error
	)
	{
		secretName = "test-secret"
		central = newCentral()
		validate = func(dataMap types.SecretDataMap, b bool) error {
			return nil
		}
		generate = func(dataMap types.SecretDataMap) (types.SecretDataMap, error) {
			return types.SecretDataMap{
				"test": []byte("test"),
			}, nil
		}
	}

	for _, tt := range tests {

		exists := tt.state != "secretDoesNotExist"
		hasOwnerReference := strings.Contains(tt.state, "hasOwnerRef")
		hasLabel := strings.Contains(tt.state, "hasLabel")
		hasWrongLabel := strings.Contains(tt.state, "hasWrongLabel")

		t.Run("strategy:"+string(tt.strategy)+",state:"+tt.state, func(t *testing.T) {

			var objects []ctrlClient.Object
			if exists {
				secret := &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: testutils.TestNamespace,
						Labels:    map[string]string{},
					},
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}
				if hasOwnerReference {
					secret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(central, platform.CentralGVK)})
				}
				if hasLabel {
					secret.Labels = map[string]string{
						managedByOperatorLabel: managedByOperatorValue,
					}
				}
				if hasWrongLabel {
					secret.Labels = map[string]string{
						managedByOperatorLabel: "wrong",
					}
				}
				objects = append(objects, secret)
			}
			client := fake.NewClientBuilder().WithObjects(objects...).Build()
			ctx := context.Background()
			err := NewSecretReconciliator(client, central, tt.strategy).EnsureSecret(ctx, secretName, validate, generate)
			require.NoError(t, err)

			var secret v1.Secret
			require.NoError(t, client.Get(ctx, k8sTypes.NamespacedName{Name: secretName, Namespace: testutils.TestNamespace}, &secret))

			if tt.expectedLabel {
				assert.Equal(t, managedByOperatorValue, secret.Labels[managedByOperatorLabel])
			} else {
				assert.NotEqual(t, managedByOperatorValue, secret.Labels[managedByOperatorLabel])
			}

			if tt.expectedOwnerRef {
				assert.True(t, metav1.IsControlledBy(&secret, central))
			} else {
				assert.False(t, metav1.IsControlledBy(&secret, central))
			}
		})

	}
}

func newCentral() *platform.Central {
	return &platform.Central{
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
}
