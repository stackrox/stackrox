package extensions

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	emptyStorageClass = ""
	testPVCName       = "stackrox-db-test"
)

type pvcReconciliationTestCase struct {
	Central      *platform.Central
	ExistingPVCs []*corev1.PersistentVolumeClaim
	Delete       bool

	ExpectedPVCs  map[string]pvcVerifyFunc
	ExpectedError string
}

type pvcVerifyFunc func(t *testing.T, pvc *corev1.PersistentVolumeClaim)

func verifyMultiple(funcs ...pvcVerifyFunc) pvcVerifyFunc {
	return func(t *testing.T, pvc *corev1.PersistentVolumeClaim) {
		for _, fn := range funcs {
			fn(t, pvc)
		}
	}
}

func ownedBy(central *platform.Central) pvcVerifyFunc {
	return func(t *testing.T, pvc *corev1.PersistentVolumeClaim) {
		require.NotNil(t, pvc)
		assert.True(t, metav1.IsControlledBy(pvc, central))
	}
}

func withSize(size resource.Quantity) pvcVerifyFunc {
	return func(t *testing.T, pvc *corev1.PersistentVolumeClaim) {
		require.NotNil(t, pvc)
		assert.Equal(t, size.String(), pvc.Spec.Resources.Requests.Storage().String())
	}
}

func withStorageClass(storageClass string) pvcVerifyFunc {
	return func(t *testing.T, pvc *corev1.PersistentVolumeClaim) {
		require.NotNil(t, pvc)
		assert.Equal(t, storageClass, pointer.StringPtrDerefOr(pvc.Spec.StorageClassName, ""))
	}
}

func pvcNotCreatedVerifier(t *testing.T, pvc *corev1.PersistentVolumeClaim) {
	assert.Nil(t, pvc, "PVC should not be created if hostpath is given")
}

func notOwnedBy(central *platform.Central) pvcVerifyFunc {
	return func(t *testing.T, pvc *corev1.PersistentVolumeClaim) {
		require.NotNil(t, pvc)
		assert.False(t, metav1.IsControlledBy(pvc, central))

	}
}

func TestReconcilePVCExtension(t *testing.T) {
	removedCentral := makeCentral(nil)
	emptyNotDeletedCentral := makeCentral(nil)
	deleteHostPathCentral := makeCentral(&platform.Persistence{HostPath: makeHostPathSpec("/tmp/path")})

	changedPVCConfigCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			Size:             pointer.StringPtr("500Gi"),
			StorageClassName: pointer.StringPtr("new-storage-class"),
			ClaimName:        pointer.StringPtr(testPVCName),
		},
	})
	changedPVCNameCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			ClaimName: pointer.StringPtr(testPVCName),
			Size:      pointer.StringPtr("500Gi"),
		},
	})
	referencedPVCCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			ClaimName: pointer.StringPtr(testPVCName),
		},
	})
	pvcShouldCreateWithConfigCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			ClaimName:        pointer.StringPtr(testPVCName),
			Size:             pointer.StringPtr("50Gi"),
			StorageClassName: pointer.StringPtr("test-storage-class"),
		},
	})
	notOwnedPVCConfigChangeCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			Size:             pointer.StringPtr("500Gi"),
			StorageClassName: pointer.StringPtr("new-storage-class"),
		},
	})

	cases := map[string]pvcReconciliationTestCase{
		"empty-state-create-new-default-pvc": {
			Central:      emptyNotDeletedCentral,
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: verifyMultiple(ownedBy(emptyNotDeletedCentral), withSize(defaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},

		"given-hostpath-and-pvc-should-return-error": {
			Central: makeCentral(&platform.Persistence{
				HostPath:              makeHostPathSpec("/tmp/hostpath"),
				PersistentVolumeClaim: &platform.PersistentVolumeClaim{},
			}),
			ExistingPVCs:  nil,
			ExpectedError: "invalid persistence configuration, either hostPath oder persistentVolumeClaim must be set, not both",
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: pvcNotCreatedVerifier,
			},
		},

		"given-hostpath-should-not-create-pvc": {
			Central:      makeCentral(&platform.Persistence{HostPath: makeHostPathSpec("/tmp/hostpath")}),
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: pvcNotCreatedVerifier,
			},
		},

		"given-pvc-should-create-pvc-with-config": {
			Central:      pvcShouldCreateWithConfigCentral,
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(pvcShouldCreateWithConfigCentral), withSize(resource.MustParse("50Gi")), withStorageClass("test-storage-class")),
			},
		},

		"existing-pvc-should-be-reconciled": {
			Central:      changedPVCConfigCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(changedPVCConfigCentral, testPVCName, defaultPVCSize, emptyStorageClass)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(changedPVCConfigCentral), withSize(resource.MustParse("500Gi")), withStorageClass("new-storage-class")),
			},
		},

		"only-one-pvc-with-owner-ref-is-allowed": {
			Central:       changedPVCNameCentral,
			ExistingPVCs:  []*corev1.PersistentVolumeClaim{makePVC(changedPVCNameCentral, DefaultPVCName, defaultPVCSize, emptyStorageClass)},
			ExpectedError: `Could not create PVC "stackrox-db-test" because the operator can only manage 1 PVC for Central. To fix this either reference a manually created PVC or remove the OwnerReference of the "stackrox-db" PVC.`,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: verifyMultiple(ownedBy(changedPVCNameCentral)),
				testPVCName:    pvcNotCreatedVerifier,
			},
		},

		"given-pvc-without-owner-ref-can-be-referenced": {
			Central:      emptyNotDeletedCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: DefaultPVCName, Namespace: "stackrox"}}},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: verifyMultiple(notOwnedBy(emptyNotDeletedCentral)),
			},
		},

		"config-changes-on-pvcs-not-owned-by-the-operator-should-fail": {
			Central:       notOwnedPVCConfigChangeCentral,
			ExistingPVCs:  []*corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: DefaultPVCName, Namespace: "stackrox"}}},
			ExpectedError: `Failed reconciling PVC "stackrox-db". Please remove the storageClassName and size properties from your spec, or change the name to allow the operator to create a new one with a different name.`,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: verifyMultiple(notOwnedBy(notOwnedPVCConfigChangeCentral)),
			},
		},

		"change-claim-name-to-a-not-operator-managed-pvc-should-be-reconciled": {
			Central: referencedPVCCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(referencedPVCCentral, DefaultPVCName, defaultPVCSize, emptyStorageClass),
				{ObjectMeta: metav1.ObjectMeta{
					Name:      testPVCName,
					Namespace: "stackrox",
				}},
			},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: verifyMultiple(ownedBy(referencedPVCCentral)),
				testPVCName:    verifyMultiple(notOwnedBy(referencedPVCCentral)),
			},
		},

		"delete-central-with-active-hostpath-and-existing-owned-pvcs-should-remove-owner-refs": {
			Central: deleteHostPathCentral,
			Delete:  true,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(deleteHostPathCentral, DefaultPVCName, defaultPVCSize, emptyStorageClass),
				makePVC(deleteHostPathCentral, testPVCName, defaultPVCSize, emptyStorageClass),
			},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: notOwnedBy(deleteHostPathCentral),
				testPVCName:    notOwnedBy(deleteHostPathCentral),
			},
		},

		"delete-central-should-remove-owner-reference": {
			Central:      removedCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(removedCentral, DefaultPVCName, defaultPVCSize, emptyStorageClass)},
			Delete:       true,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: verifyMultiple(notOwnedBy(removedCentral)),
			},
		},

		"reconciliation-should-fail-with-multiple-operator-owned-PVCs": {
			Central: emptyNotDeletedCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(emptyNotDeletedCentral, DefaultPVCName, defaultPVCSize, emptyStorageClass),
				makePVC(emptyNotDeletedCentral, testPVCName, defaultPVCSize, emptyStorageClass),
			},
			ExpectedError: "multiple owned PVCs were found, please remove not used ones or delete their OwnerReferences. Found PVCs: stackrox-db, stackrox-db-test: operator is only allowed to have 1 owned PVC",
		},

		"storage-class-is-not-reconciled-if-empty-storage-class-is-given": {
			Central:      emptyNotDeletedCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(emptyNotDeletedCentral, DefaultPVCName, defaultPVCSize, "storage-class-name")},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultPVCName: verifyMultiple(withStorageClass("storage-class-name"), withSize(defaultPVCSize)),
			},
		},

		"given-an-unmanaged-pvc-referencing-a-nonexisting-pvc-should-create-pvc": {
			Central:      emptyNotDeletedCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: testPVCName, Namespace: "stackrox"}}},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName:    verifyMultiple(notOwnedBy(emptyNotDeletedCentral)),
				DefaultPVCName: verifyMultiple(ownedBy(emptyNotDeletedCentral), withSize(resource.MustParse("100Gi")), withStorageClass(emptyStorageClass)),
			},
		},
	}

	for name, tc := range cases {
		testCase := tc

		t.Run(name, func(t *testing.T) {
			if testCase.Delete {
				time := metav1.Now()
				testCase.Central.DeletionTimestamp = &time
			}

			var allExisting []ctrlClient.Object
			for _, existingPVC := range testCase.ExistingPVCs {
				allExisting = append(allExisting, existingPVC)
			}

			client := fake.NewClientBuilder().WithObjects(allExisting...).Build()

			rFirstRun := newReconcilePVCExtensionRun(testCase.Central, client)
			executeAndVerify(t, testCase, rFirstRun)

			// Run it a second time to verify cluster state does not change after first reconciliation was executed
			rSecondRun := newReconcilePVCExtensionRun(testCase.Central, client)
			executeAndVerify(t, testCase, rSecondRun)
		})
	}
}

func executeAndVerify(t *testing.T, testCase pvcReconciliationTestCase, r reconcilePVCExtensionRun) {
	err := r.Execute()

	if testCase.ExpectedError == "" {
		require.NoError(t, err)
	} else {
		assert.EqualError(t, err, testCase.ExpectedError)
	}

	if len(testCase.ExpectedPVCs) == 0 {
		return
	}

	pvcsToVerify := make(map[string]pvcVerifyFunc)
	for name, pvf := range testCase.ExpectedPVCs {
		pvcsToVerify[name] = pvf
	}

	pvcList := &corev1.PersistentVolumeClaimList{}
	err = r.client.List(context.TODO(), pvcList)
	require.NoError(t, err)

	// check pvcs which should exist in cluster
	for i := range pvcList.Items {
		pvc := pvcList.Items[i]
		pvf := pvcsToVerify[pvc.GetName()]
		require.NotNilf(t, pvf, "unexpected PVC %s", pvc.GetName())
		pvf(t, &pvc)
		delete(pvcsToVerify, pvc.GetName())
	}

	// Check pvs which should not exit in cluster
	for _, pvf := range pvcsToVerify {
		pvf(t, nil)
	}
}

func makeCentral(p *platform.Persistence) *platform.Central {
	return &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID(uuid.NewV4().String()),
		},
		Spec: platform.CentralSpec{
			Central: &platform.CentralComponentSpec{
				Persistence: p,
			},
		},
	}
}

func makePVC(owner *platform.Central, name string, size resource.Quantity, storageClassName string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "stackrox",
			Name:            name,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(owner, owner.GroupVersionKind())},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: pointer.StringPtr(storageClassName),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
		},
	}

}

func newReconcilePVCExtensionRun(c *platform.Central, client ctrlClient.Client) reconcilePVCExtensionRun {
	return reconcilePVCExtensionRun{
		ctx:        context.Background(),
		namespace:  "stackrox",
		client:     client,
		centralObj: c,
		log:        logr.Discard(),
	}
}

func makeHostPathSpec(path string) *platform.HostPathSpec {
	return &platform.HostPathSpec{
		Path: &path,
	}
}
