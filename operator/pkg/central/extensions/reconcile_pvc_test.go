package extensions

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/central/common"
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
	Target       PVCTarget
	DefaultClaim string
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
		assert.True(t, metav1.IsControlledBy(pvc, central),
			"expected PVC to be owned by central %q, but its owner references were %q",
			central.UID, pvc.OwnerReferences)
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
		assert.Equal(t, storageClass, pointer.StringDeref(pvc.Spec.StorageClassName, ""))
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
	emptyNotDeletedCentralWithDB := makeCentral(nil)
	emptyNotDeletedCentralWithDB.Spec.Central.DB = &platform.CentralDBSpec{}

	pvcObsoletedAnnotation := map[string]string{
		common.CentralPVCObsoleteAnnotation: "true",
	}
	centralWithPvcObsoletedAnnotation := makeCentral(nil)
	centralWithPvcObsoletedAnnotation.Annotations = pvcObsoletedAnnotation
	centralWithPersistenceAndPvcObsoletedAnnotation := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			Size:             pointer.String("500Gi"),
			StorageClassName: pointer.String("new-storage-class"),
			ClaimName:        pointer.String(testPVCName),
		},
	})
	centralWithPersistenceAndPvcObsoletedAnnotation.Annotations = pvcObsoletedAnnotation

	externalCentralWithDB := makeCentral(nil)
	externalCentralWithDB.Spec.Central.DB = &platform.CentralDBSpec{}
	externalCentralWithDB.Spec.Central.DB.ConnectionStringOverride = pointer.String("foobar")

	deleteHostPathCentral := makeCentral(&platform.Persistence{HostPath: makeHostPathSpec("/tmp/path")})

	changedPVCConfigCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			Size:             pointer.String("500Gi"),
			StorageClassName: pointer.String("new-storage-class"),
			ClaimName:        pointer.String(testPVCName),
		},
	})
	changedPVCNameCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			ClaimName: pointer.String(testPVCName),
			Size:      pointer.String("500Gi"),
		},
	})
	referencedPVCCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			ClaimName: pointer.String(testPVCName),
		},
	})
	pvcShouldCreateWithConfigCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			ClaimName:        pointer.String(testPVCName),
			Size:             pointer.String("50Gi"),
			StorageClassName: pointer.String("test-storage-class"),
		},
	})
	notOwnedPVCConfigChangeCentral := makeCentral(&platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			Size:             pointer.String("500Gi"),
			StorageClassName: pointer.String("new-storage-class"),
		},
	})
	centralTargetAnnotations := map[string]string{
		pvcTargetLabelKey: string(PVCTargetCentral),
	}
	centralDBTargetAnnotations := map[string]string{
		pvcTargetLabelKey: string(PVCTargetCentralDB),
	}

	changedPVCConfigCentralDB := makeCentral(nil)
	changedPVCConfigCentralDB.Spec.Central.DB = &platform.CentralDBSpec{
		Persistence: &platform.DBPersistence{
			PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
				Size:             pointer.String("500Gi"),
				StorageClassName: pointer.String("new-storage-class"),
				ClaimName:        pointer.String(testPVCName),
			},
		},
	}

	cases := map[string]pvcReconciliationTestCase{
		"empty-state-not-create-new-default-pvc": {
			Central:      emptyNotDeletedCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: nil,
			ExpectedPVCs: nil,
		},
		"empty-state-keep-default-pvc": {
			Central:      emptyNotDeletedCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(emptyNotDeletedCentral, DefaultCentralPVCName, defaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(ownedBy(emptyNotDeletedCentral), withSize(defaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},
		"empty-state-obsolete-default-pvc": {
			Central:      centralWithPvcObsoletedAnnotation,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(centralWithPvcObsoletedAnnotation, DefaultCentralPVCName, defaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(notOwnedBy(emptyNotDeletedCentral), withSize(defaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},
		"given-hostpath-and-pvc-should-return-error": {
			Central: makeCentral(&platform.Persistence{
				HostPath:              makeHostPathSpec("/tmp/hostpath"),
				PersistentVolumeClaim: &platform.PersistentVolumeClaim{},
			}),
			DefaultClaim:  DefaultCentralPVCName,
			Target:        PVCTargetCentral,
			ExistingPVCs:  nil,
			ExpectedError: "invalid persistence configuration, either hostPath or persistentVolumeClaim must be set, not both",
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: pvcNotCreatedVerifier,
			},
		},

		"given-hostpath-should-not-create-pvc": {
			Central:      makeCentral(&platform.Persistence{HostPath: makeHostPathSpec("/tmp/hostpath")}),
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: pvcNotCreatedVerifier,
			},
		},

		"given-pvc-should-not-create-pvc-with-config": {
			Central:      pvcShouldCreateWithConfigCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: nil,
			ExpectedPVCs: nil,
		},

		"given-pvc-should-keep-pvc-with-config": {
			Central:      pvcShouldCreateWithConfigCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(pvcShouldCreateWithConfigCentral, testPVCName, defaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(pvcShouldCreateWithConfigCentral), withSize(resource.MustParse("50Gi")), withStorageClass("test-storage-class")),
			},
		},
		"given-pvc-should-obsolete-pvc-with-config": {
			Central:      centralWithPersistenceAndPvcObsoletedAnnotation,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(centralWithPersistenceAndPvcObsoletedAnnotation, testPVCName, defaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(notOwnedBy(pvcShouldCreateWithConfigCentral), withSize(resource.MustParse(defaultPVCSize.String())), withStorageClass(emptyStorageClass)),
			},
		},
		"existing-pvc-should-be-reconciled-with-no-annotation": {
			Central:      changedPVCConfigCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(changedPVCConfigCentral, testPVCName, defaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(changedPVCConfigCentral), withSize(resource.MustParse("500Gi")), withStorageClass("new-storage-class")),
			},
		},
		"existing-pvc-should-be-reconciled-with-annotation": {
			Central:      changedPVCConfigCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(changedPVCConfigCentral, testPVCName, defaultPVCSize, emptyStorageClass, centralTargetAnnotations)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(changedPVCConfigCentral), withSize(resource.MustParse("500Gi")), withStorageClass("new-storage-class")),
			},
		},

		"only-one-pvc-with-owner-ref-is-allowed": {
			Central:       changedPVCNameCentral,
			DefaultClaim:  DefaultCentralPVCName,
			Target:        PVCTargetCentral,
			ExistingPVCs:  []*corev1.PersistentVolumeClaim{makePVC(changedPVCNameCentral, DefaultCentralPVCName, defaultPVCSize, emptyStorageClass, centralTargetAnnotations)},
			ExpectedError: `Could not create PVC "stackrox-db-test" because the operator can only manage 1 PVC for central. To fix this either reference a manually created PVC or remove the OwnerReference of the "stackrox-db" PVC.`,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(ownedBy(changedPVCNameCentral)),
				testPVCName:           pvcNotCreatedVerifier,
			},
		},

		"given-pvc-without-owner-ref-can-be-referenced": {
			Central:      emptyNotDeletedCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: DefaultCentralPVCName, Namespace: "stackrox"}}},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(notOwnedBy(emptyNotDeletedCentral)),
			},
		},

		"config-changes-on-pvcs-not-owned-by-the-operator-should-fail": {
			Central:       notOwnedPVCConfigChangeCentral,
			DefaultClaim:  DefaultCentralPVCName,
			Target:        PVCTargetCentral,
			ExistingPVCs:  []*corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: DefaultCentralPVCName, Namespace: "stackrox"}}},
			ExpectedError: `Failed reconciling PVC "stackrox-db". Please remove the storageClassName and size properties from your spec, or change the name to allow the operator to create a new one with a different name.`,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(notOwnedBy(notOwnedPVCConfigChangeCentral)),
			},
		},

		"change-claim-name-to-a-not-operator-managed-pvc-should-be-reconciled": {
			Central:      referencedPVCCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(referencedPVCCentral, DefaultCentralPVCName, defaultPVCSize, emptyStorageClass, centralTargetAnnotations),
				{ObjectMeta: metav1.ObjectMeta{
					Name:      testPVCName,
					Namespace: "stackrox",
				}},
			},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(ownedBy(referencedPVCCentral)),
				testPVCName:           verifyMultiple(notOwnedBy(referencedPVCCentral)),
			},
		},

		"delete-central-with-active-hostpath-and-existing-owned-pvcs-should-remove-owner-refs": {
			Central:      deleteHostPathCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			Delete:       true,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(deleteHostPathCentral, DefaultCentralPVCName, defaultPVCSize, emptyStorageClass, centralTargetAnnotations),
				makePVC(deleteHostPathCentral, testPVCName, defaultPVCSize, emptyStorageClass, centralTargetAnnotations),
			},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: notOwnedBy(deleteHostPathCentral),
				testPVCName:           notOwnedBy(deleteHostPathCentral),
			},
		},

		"delete-central-should-remove-owner-reference": {
			Central:      removedCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(removedCentral, DefaultCentralPVCName, defaultPVCSize, emptyStorageClass, centralTargetAnnotations)},
			Delete:       true,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(notOwnedBy(removedCentral)),
			},
		},

		"reconciliation-should-fail-with-multiple-operator-owned-PVCs": {
			Central:      emptyNotDeletedCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(emptyNotDeletedCentral, DefaultCentralPVCName, defaultPVCSize, emptyStorageClass, centralTargetAnnotations),
				makePVC(emptyNotDeletedCentral, testPVCName, defaultPVCSize, emptyStorageClass, centralTargetAnnotations),
			},
			ExpectedError: "multiple owned PVCs were found for central, please remove not used ones or delete their OwnerReferences. Found PVCs: stackrox-db, stackrox-db-test: operator is only allowed to have 1 owned PVC",
		},

		"storage-class-is-not-reconciled-if-empty-storage-class-is-given": {
			Central:      emptyNotDeletedCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(emptyNotDeletedCentral, DefaultCentralPVCName, defaultPVCSize, "storage-class-name", centralTargetAnnotations)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(withStorageClass("storage-class-name"), withSize(defaultPVCSize)),
			},
		},

		"given-an-unmanaged-pvc-referencing-a-nonexisting-pvc-should-create-pvc": {
			Central:      emptyNotDeletedCentral,
			DefaultClaim: DefaultCentralPVCName,
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: testPVCName, Namespace: "stackrox"}}},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(notOwnedBy(emptyNotDeletedCentral)),
			},
		},

		"external central-db provided and no pvc should be created": {
			Central:      externalCentralWithDB,
			DefaultClaim: DefaultCentralDBPVCName,
			Target:       PVCTargetCentralDB,
			ExistingPVCs: nil,
			ExpectedPVCs: nil,
		},

		"central-db-empty-state-not-create-new-default-pvc": {
			Central:      emptyNotDeletedCentralWithDB,
			DefaultClaim: DefaultCentralDBPVCName,
			Target:       PVCTargetCentralDB,
			ExistingPVCs: nil,
			ExpectedPVCs: nil,
		},

		"central-db-empty-state-create-new-default-pvc-no-annotation-pvc": {
			Central:      emptyNotDeletedCentralWithDB,
			DefaultClaim: DefaultCentralDBPVCName,
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(emptyNotDeletedCentralWithDB, DefaultCentralPVCName, defaultPVCSize, emptyStorageClass, nil),
			},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName:   verifyMultiple(ownedBy(emptyNotDeletedCentralWithDB), withSize(defaultPVCSize), withStorageClass(emptyStorageClass)),
				DefaultCentralDBPVCName: verifyMultiple(ownedBy(emptyNotDeletedCentralWithDB), withSize(defaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},

		"central-db-empty-state-create-new-default-pvc-with-central-annotation-pvc": {
			Central:      emptyNotDeletedCentralWithDB,
			DefaultClaim: DefaultCentralDBPVCName,
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(emptyNotDeletedCentralWithDB, DefaultCentralPVCName, defaultPVCSize, emptyStorageClass, centralTargetAnnotations),
			},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName:   verifyMultiple(ownedBy(emptyNotDeletedCentralWithDB), withSize(defaultPVCSize), withStorageClass(emptyStorageClass)),
				DefaultCentralDBPVCName: verifyMultiple(ownedBy(emptyNotDeletedCentralWithDB), withSize(defaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},
		"central-db-existing-pvc-should-be-reconciled-with-annotation": {
			Central:      changedPVCConfigCentralDB,
			DefaultClaim: DefaultCentralDBPVCName,
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(changedPVCConfigCentralDB, testPVCName, defaultPVCSize, emptyStorageClass, centralDBTargetAnnotations)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(changedPVCConfigCentralDB), withSize(resource.MustParse("500Gi")), withStorageClass("new-storage-class")),
			},
		},
		"central-pvc-should-not-lose-owner-refs": {
			Central:      emptyNotDeletedCentralWithDB,
			DefaultClaim: DefaultCentralDBPVCName,
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(emptyNotDeletedCentralWithDB, testPVCName, defaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBPVCName: verifyMultiple(ownedBy(emptyNotDeletedCentralWithDB), withSize(defaultPVCSize), withStorageClass(emptyStorageClass)),
				testPVCName:             ownedBy(emptyNotDeletedCentralWithDB),
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

			rFirstRun := newReconcilePVCExtensionRun(testCase, client)
			executeAndVerify(t, testCase, rFirstRun)

			// Run it a second time to verify cluster state does not change after first reconciliation was executed
			rSecondRun := newReconcilePVCExtensionRun(testCase, client)
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

	// Check pvcs which should exist in cluster.
	for i := range pvcList.Items {
		pvc := pvcList.Items[i]
		pvf := pvcsToVerify[pvc.GetName()]
		require.NotNilf(t, pvf, "unexpected PVC %s", pvc.GetName())
		pvf(t, &pvc)
		delete(pvcsToVerify, pvc.GetName())
	}

	// Check pvs which should not exist in cluster.
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

func makePVC(owner *platform.Central, name string, size resource.Quantity, storageClassName string, _ map[string]string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "stackrox",
			Name:            name,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(owner, owner.GroupVersionKind())},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: pointer.String(storageClassName),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
		},
	}
}

func newReconcilePVCExtensionRun(testCase pvcReconciliationTestCase, client ctrlClient.Client) reconcilePVCExtensionRun {
	return reconcilePVCExtensionRun{
		ctx:              context.Background(),
		namespace:        "stackrox",
		client:           client,
		centralObj:       testCase.Central,
		target:           testCase.Target,
		defaultClaimName: testCase.DefaultClaim,
		persistence:      getPersistenceByTarget(testCase.Central, testCase.Target),
		log:              logr.Discard(),
	}
}

func makeHostPathSpec(path string) *platform.HostPathSpec {
	return &platform.HostPathSpec{
		Path: &path,
	}
}
