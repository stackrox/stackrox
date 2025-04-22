package extensions

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/central/common"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	emptyStorageClass = ""
	testPVCName       = "central-db-test"
)

// DefaultPVCValues specifies a set of default values used when reconciling the PVC.
type DefaultPVCValues struct {
	ClaimName string
	Size      resource.Quantity
}

type pvcReconciliationTestCase struct {
	Central      *platform.Central
	Target       PVCTarget
	Defaults     DefaultPVCValues
	ExistingPVCs []*corev1.PersistentVolumeClaim

	ExpectedPVCs  map[string]pvcVerifyFunc
	ExpectedError string

	DefaultStorageClass bool
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
	emptyCentral := makeCentral(nil)

	pvcObsoletedAnnotation := map[string]string{
		common.CentralPVCObsoleteAnnotation: "true",
	}
	emptyCentralWithPvcObsoletedAnnotation := makeCentral(nil)
	emptyCentralWithPvcObsoletedAnnotation.Annotations = pvcObsoletedAnnotation

	externalCentralWithDB := makeCentral(nil)
	externalCentralWithDB.Spec.Central.DB.ConnectionStringOverride = pointer.String("foobar")

	changedPVCNameCentral := makeCentral(&platform.DBPersistence{
		PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
			ClaimName: pointer.String(testPVCName),
		},
	})
	changedPVCSizeAndStorageClassCentral := makeCentral(&platform.DBPersistence{
		PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
			Size:             pointer.String("500Gi"),
			StorageClassName: pointer.String("new-storage-class"),
		},
	})
	centralTargetLabels := map[string]string{
		pvcTargetLabelKey: string(PVCTargetCentral),
	}
	centralDBTargetLabels := map[string]string{
		pvcTargetLabelKey: string(PVCTargetCentralDB),
	}

	changedPVCConfigCentral := makeCentral(nil)
	changedPVCConfigCentral.Spec.Central.DB = &platform.CentralDBSpec{
		Persistence: &platform.DBPersistence{
			PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
				Size:             pointer.String("500Gi"),
				StorageClassName: pointer.String("new-storage-class"),
				ClaimName:        pointer.String(testPVCName),
			},
		},
	}

	deletedCentral := makeCentral(nil)
	deleteTime := metav1.Now()
	deletedCentral.DeletionTimestamp = &deleteTime

	cases := map[string]pvcReconciliationTestCase{
		"empty-state-not-create-new-default-central-pvc": {
			DefaultStorageClass: true,
			Central:             emptyCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentral,
			ExistingPVCs: nil,
			ExpectedPVCs: nil,
		},
		"empty-state-obsolete-default-central-pvc": {
			DefaultStorageClass: true,
			Central:             emptyCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(emptyCentral, DefaultCentralPVCName, DefaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(notOwnedBy(emptyCentral), withSize(DefaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},
		"empty-state-obsolete-default-central-pvc-with-obsolete-annotation": {
			DefaultStorageClass: true,
			Central:             emptyCentralWithPvcObsoletedAnnotation,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(emptyCentralWithPvcObsoletedAnnotation, DefaultCentralPVCName, DefaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: verifyMultiple(notOwnedBy(emptyCentral), withSize(DefaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},
		"central-pvc-should-lose-owner-refs": {
			DefaultStorageClass: true,
			Central:             emptyCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentral,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(emptyCentral, DefaultCentralPVCName, DefaultPVCSize, emptyStorageClass, centralTargetLabels)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName: notOwnedBy(emptyCentral),
			},
		},
		"given-hostpath-should-not-create-default-central-db-pvc": {
			DefaultStorageClass: true,
			Central:             makeCentral(&platform.DBPersistence{HostPath: makeHostPathSpec("/tmp/hostpath")}),
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBPVCName: pvcNotCreatedVerifier,
			},
		},

		"given-pvc-should-create-default-central-db-pvc-with-config": {
			DefaultStorageClass: true,
			Central:             changedPVCConfigCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(changedPVCConfigCentral), withSize(resource.MustParse("500Gi")), withStorageClass("new-storage-class")),
			},
		},

		"given-pvc-should-keep-central-db-pvc-with-config": {
			DefaultStorageClass: true,
			Central:             changedPVCConfigCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(changedPVCConfigCentral, testPVCName, DefaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(changedPVCConfigCentral), withSize(resource.MustParse("500Gi")), withStorageClass("new-storage-class")),
			},
		},

		"existing-pvc-should-be-reconciled-with-no-labels": {
			DefaultStorageClass: true,
			Central:             changedPVCConfigCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(changedPVCConfigCentral, testPVCName, DefaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(changedPVCConfigCentral), withSize(resource.MustParse("500Gi")), withStorageClass("new-storage-class")),
			},
		},
		"existing-pvc-should-be-reconciled-with-labels": {
			DefaultStorageClass: true,
			Central:             changedPVCConfigCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(changedPVCConfigCentral, testPVCName, DefaultPVCSize, emptyStorageClass, centralDBTargetLabels)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(changedPVCConfigCentral), withSize(resource.MustParse("500Gi")), withStorageClass("new-storage-class")),
			},
		},

		"only-one-pvc-with-owner-ref-is-allowed": {
			DefaultStorageClass: true,
			Central:             changedPVCNameCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:        PVCTargetCentralDB,
			ExistingPVCs:  []*corev1.PersistentVolumeClaim{makePVC(changedPVCNameCentral, DefaultCentralDBPVCName, DefaultPVCSize, emptyStorageClass, centralDBTargetLabels)},
			ExpectedError: `Could not create PVC "central-db-test" because the operator can only manage 1 PVC for central-db. To fix this either reference a manually created PVC or remove the OwnerReference of the "central-db" PVC.`,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBPVCName: verifyMultiple(ownedBy(changedPVCNameCentral)),
				testPVCName:             pvcNotCreatedVerifier,
			},
		},

		"config-changes-on-pvcs-not-owned-by-the-operator-should-fail": {
			DefaultStorageClass: true,
			Central:             changedPVCSizeAndStorageClassCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:        PVCTargetCentralDB,
			ExistingPVCs:  []*corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: DefaultCentralDBPVCName, Namespace: "stackrox"}}},
			ExpectedError: `Failed reconciling PVC "central-db". Please remove the storageClassName and size properties from your spec, or change the name to allow the operator to create a new one with a different name.`,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBPVCName: verifyMultiple(notOwnedBy(changedPVCSizeAndStorageClassCentral)),
			},
		},

		"change-claim-name-to-a-not-operator-managed-pvc-should-be-reconciled": {
			DefaultStorageClass: true,
			Central:             changedPVCNameCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target: PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(changedPVCNameCentral, DefaultCentralDBPVCName, DefaultPVCSize, emptyStorageClass, centralDBTargetLabels),
				{ObjectMeta: metav1.ObjectMeta{
					Name:      testPVCName,
					Namespace: "stackrox",
				}},
			},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBPVCName: verifyMultiple(ownedBy(changedPVCNameCentral)),
				testPVCName:             verifyMultiple(notOwnedBy(changedPVCNameCentral)),
			},
		},

		"delete-central-should-remove-owner-reference": {
			DefaultStorageClass: true,
			Central:             deletedCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(deletedCentral, DefaultCentralDBPVCName, DefaultPVCSize, emptyStorageClass, centralDBTargetLabels)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBPVCName: verifyMultiple(notOwnedBy(deletedCentral)),
			},
		},

		"external central-db provided and no pvc should be created": {
			DefaultStorageClass: true,
			Central:             externalCentralWithDB,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: nil,
			ExpectedPVCs: nil,
		},
		"central-db-empty-state-create-new-default-central-db-pvc": {
			DefaultStorageClass: true,
			Central:             emptyCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBPVCName: verifyMultiple(ownedBy(emptyCentral), withSize(DefaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},

		"central-db-empty-state-create-new-default-pvc-no-labels-pvc": {
			DefaultStorageClass: true,
			Central:             emptyCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target: PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(emptyCentral, DefaultCentralPVCName, DefaultPVCSize, emptyStorageClass, nil),
			},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName:   verifyMultiple(ownedBy(emptyCentral), withSize(DefaultPVCSize), withStorageClass(emptyStorageClass)),
				DefaultCentralDBPVCName: verifyMultiple(ownedBy(emptyCentral), withSize(DefaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},

		"central-db-empty-state-create-new-default-pvc-with-central-annotation-pvc": {
			DefaultStorageClass: true,
			Central:             emptyCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target: PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{
				makePVC(emptyCentral, DefaultCentralPVCName, DefaultPVCSize, emptyStorageClass, centralTargetLabels),
			},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralPVCName:   verifyMultiple(ownedBy(emptyCentral), withSize(DefaultPVCSize), withStorageClass(emptyStorageClass)),
				DefaultCentralDBPVCName: verifyMultiple(ownedBy(emptyCentral), withSize(DefaultPVCSize), withStorageClass(emptyStorageClass)),
			},
		},
		"central-db-existing-pvc-should-be-reconciled-with-labels": {
			DefaultStorageClass: true,
			Central:             changedPVCConfigCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(changedPVCConfigCentral, testPVCName, DefaultPVCSize, emptyStorageClass, centralDBTargetLabels)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				testPVCName: verifyMultiple(ownedBy(changedPVCConfigCentral), withSize(resource.MustParse("500Gi")), withStorageClass("new-storage-class")),
			},
		},
		"central-pvc-should-not-lose-owner-refs": {
			DefaultStorageClass: true,
			Central:             emptyCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBPVCName,
				Size:      DefaultPVCSize,
			},
			Target:       PVCTargetCentralDB,
			ExistingPVCs: []*corev1.PersistentVolumeClaim{makePVC(emptyCentral, testPVCName, DefaultPVCSize, emptyStorageClass, nil)},
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBPVCName: verifyMultiple(ownedBy(emptyCentral), withSize(DefaultPVCSize), withStorageClass(emptyStorageClass)),
				testPVCName:             ownedBy(emptyCentral),
			},
		},

		// Test that a backup volume is created, happy path
		"central-db-empty-state-create-new-default-central-db-backup-pvc": {
			DefaultStorageClass: true,
			Central:             emptyCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBBackupPVCName,
				Size:      DefaultBackupPVCSize,
			},
			Target:       PVCTargetCentralDBBackup,
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBBackupPVCName: verifyMultiple(
					ownedBy(emptyCentral),
					withSize(DefaultBackupPVCSize),
					withStorageClass(emptyStorageClass)),
			},
		},

		// Test that in absense of a default storage class and explicit storage
		// class in the pvc config, no backup volume will be created
		"central-db-no-default-storage-class-no-pvc": {
			DefaultStorageClass: false,
			Central:             emptyCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBBackupPVCName,
				Size:      DefaultBackupPVCSize,
			},
			Target:       PVCTargetCentralDBBackup,
			ExistingPVCs: nil,
			ExpectedPVCs: nil,
		},

		// Test that an expicitely specified storage class will be used to
		// create a backup volume, even if there is no default storage class.
		// As a side effect, verify that the size is set correctly (it should
		// be double as much as the data volume).
		"central-db-custom-storage-class-no-pvc": {
			DefaultStorageClass: false,
			Central:             changedPVCSizeAndStorageClassCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBBackupPVCName,
				Size:      DefaultBackupPVCSize,
			},
			Target:       PVCTargetCentralDBBackup,
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				DefaultCentralDBBackupPVCName: verifyMultiple(
					ownedBy(changedPVCSizeAndStorageClassCentral),
					withSize(resource.MustParse("1000Gi")),
					withStorageClass("new-storage-class")),
			},
		},

		// Verify that the backup volume ClaimName is defined from the main
		// data volume ClaimName
		"central-db-changed-pvc-claim-name": {
			DefaultStorageClass: true,
			Central:             changedPVCNameCentral,
			Defaults: DefaultPVCValues{
				ClaimName: DefaultCentralDBBackupPVCName,
				Size:      DefaultBackupPVCSize,
			},
			Target:       PVCTargetCentralDBBackup,
			ExistingPVCs: nil,
			ExpectedPVCs: map[string]pvcVerifyFunc{
				fmt.Sprintf("%s-backup", testPVCName): verifyMultiple(
					ownedBy(changedPVCNameCentral),
					withSize(DefaultBackupPVCSize),
					withStorageClass(emptyStorageClass)),
			},
		},
	}

	for name, tc := range cases {
		testCase := tc

		t.Run(name, func(t *testing.T) {
			var allExisting []ctrlClient.Object
			for _, existingPVC := range testCase.ExistingPVCs {
				allExisting = append(allExisting, existingPVC)
			}

			// Add the default storage class if requested
			if tc.DefaultStorageClass {
				allExisting = append(allExisting, &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "stackrox",
						Name:      "new-storage-class",
						Annotations: map[string]string{
							utils.DefaultStorageClassAnnotationKey: "true",
						},
					},
				})
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

func makeCentral(p *platform.DBPersistence) *platform.Central {
	return &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID(uuid.NewV4().String()),
		},
		Spec: platform.CentralSpec{
			Central: &platform.CentralComponentSpec{
				DB: &platform.CentralDBSpec{
					Persistence: p,
				},
			},
		},
	}
}

func makePVC(owner *platform.Central, name string, size resource.Quantity, storageClassName string, labels map[string]string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "stackrox",
			Name:            name,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(owner, owner.GroupVersionKind())},
			Labels:          labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: pointer.String(storageClassName),
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
		},
	}
}

func newReconcilePVCExtensionRun(testCase pvcReconciliationTestCase, client ctrlClient.Client) reconcilePVCExtensionRun {
	persistence, _ := getPersistenceByTarget(testCase.Central, testCase.Target, logr.Discard())

	return reconcilePVCExtensionRun{
		ctx:              context.Background(),
		namespace:        "stackrox",
		client:           client,
		centralObj:       testCase.Central,
		target:           testCase.Target,
		defaultClaimName: testCase.Defaults.ClaimName,
		defaultClaimSize: testCase.Defaults.Size,
		persistence:      persistence,
		log:              logr.Discard(),
	}
}

func makeHostPathSpec(path string) *platform.HostPathSpec {
	return &platform.HostPathSpec{
		Path: &path,
	}
}
