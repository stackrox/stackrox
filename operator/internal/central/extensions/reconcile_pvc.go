package extensions

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	utils "github.com/stackrox/rox/operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultCentralPVCName is the default name for Central PVC
	DefaultCentralPVCName = "stackrox-db"
	// DefaultCentralDBPVCName is the default name for Central DB PVC
	DefaultCentralDBPVCName = "central-db"

	pvcTargetLabelKey = "target.pvc.stackrox.io"
)

// PVCTarget specifies which deployment should attach the PVC
type PVCTarget string

const (
	// PVCTargetCentral is for any PVC that would be attached to the Central deployment
	PVCTargetCentral PVCTarget = "central"

	// PVCTargetCentralDB is for any PVC that would be attached to the Central DB deployment
	PVCTargetCentralDB PVCTarget = "central-db"
)

var (
	errMultipleOwnedPVCs = errors.New("operator is only allowed to have 1 owned PVC")

	defaultPVCSize = resource.MustParse("100Gi")
)

func convertDBPersistenceToPersistence(p *platform.DBPersistence) *platform.Persistence {
	if p == nil {
		return nil
	}
	if p.HostPath != nil {
		return &platform.Persistence{
			HostPath: p.HostPath,
		}
	}
	pvc := p.GetPersistentVolumeClaim()
	if pvc == nil {
		return &platform.Persistence{}
	}
	return &platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			ClaimName:        pvc.ClaimName,
			Size:             pvc.Size,
			StorageClassName: pvc.StorageClassName,
		},
	}
}

// getPersistenceByTarget retrieves the persistence configuration for the given PVC target (either PVCTargetCentral, the
// embedded persistent volume on which RocksDB is stored, or PVCTargetCentralDB, the persistent volume that serves as
// the backing store for the central-db PostgreSQL database).
// A nil return value indicates that no persistent volume should be provisioned for the respective target.
func getPersistenceByTarget(central *platform.Central, target PVCTarget) *platform.Persistence {
	switch target {
	case PVCTargetCentral:
		return nil
	case PVCTargetCentralDB:
		if !central.Spec.Central.ShouldManageDB() {
			return nil
		}
		dbPersistence := central.Spec.Central.GetDB().GetPersistence()
		if dbPersistence == nil {
			dbPersistence = &platform.DBPersistence{}
		}
		return convertDBPersistenceToPersistence(dbPersistence)
	default:
		panic(errors.Errorf("unknown pvc target %q", target))
	}
}

// ReconcilePVCExtension reconciles PVCs created by the operator. The PVC is not managed by a Helm chart
// because if a user uninstalls StackRox, it should keep the data, preventing to unintentionally erasing data.
// On uninstall the owner reference is removed from the PVC objects.
func ReconcilePVCExtension(client ctrlClient.Client, direct ctrlClient.Reader, target PVCTarget, defaultClaimName string) extensions.ReconcileExtension {
	fn := func(ctx context.Context, central *platform.Central, client ctrlClient.Client, direct ctrlClient.Reader, _ func(statusFunc updateStatusFunc), log logr.Logger) error {
		persistence := getPersistenceByTarget(central, target)
		return reconcilePVC(ctx, central, persistence, target, defaultClaimName, client, log)
	}
	return wrapExtension(fn, client, direct)
}

func reconcilePVC(ctx context.Context, central *platform.Central, persistence *platform.Persistence, target PVCTarget, defaultClaimName string, client ctrlClient.Client, log logr.Logger) error {
	ext := reconcilePVCExtensionRun{
		ctx:              ctx,
		namespace:        central.GetNamespace(),
		client:           client,
		centralObj:       central,
		persistence:      persistence,
		target:           target,
		defaultClaimName: defaultClaimName,
		log:              log.WithValues("pvcReconciler", target),
	}

	return ext.Execute()
}

type reconcilePVCExtensionRun struct {
	ctx              context.Context
	namespace        string
	client           ctrlClient.Client
	centralObj       *platform.Central
	persistence      *platform.Persistence
	defaultClaimName string
	target           PVCTarget
	log              logr.Logger
}

func (r *reconcilePVCExtensionRun) Execute() error {
	if r.centralObj.DeletionTimestamp != nil || r.target == PVCTargetCentral || r.persistence == nil {
		return r.handleDelete()
	}

	if r.persistence.GetHostPath() != "" {
		if r.persistence.GetPersistentVolumeClaim() != nil {
			return errors.New("invalid persistence configuration, either hostPath or persistentVolumeClaim must be set, not both")
		}
		return nil
	}

	pvcConfig := r.persistence.GetPersistentVolumeClaim()
	if pvcConfig == nil {
		pvcConfig = &platform.PersistentVolumeClaim{}
	}

	claimName := pointer.StringDeref(pvcConfig.ClaimName, r.defaultClaimName)
	claimNameList := []string{claimName, fmt.Sprintf("%s-backup", claimName)}

	ownedPVCList, err := r.getOwnedPVCsForCurrentTarget()
	if err != nil {
		return err
	}
	if ownedPVCList != nil {
		for _, ownedPVC := range ownedPVCList {
			// Note that originally we were checking for pvc != nil here. It's
			// not clear why should we do that, so this condition was excluded.
			if !slices.Contains(claimNameList, ownedPVC.GetName()) {
				return errors.Errorf("Could not create PVC %q because the "+
					"operator can only manage one set of PVC (data and backup) "+
					"for %s. To fix this either reference a manually created "+
					"PVC or remove the OwnerReference of the %q PVC.",
					claimName, r.target, ownedPVC.GetName())
			}
		}
	}

	// Handle reconciliation for every PVC with the same configuration
	for _, pvcName := range claimNameList {
		key := ctrlClient.ObjectKey{Namespace: r.namespace, Name: pvcName}
		pvc := &corev1.PersistentVolumeClaim{}
		if err := r.client.Get(r.ctx, key, pvc); err != nil {
			if !apiErrors.IsNotFound(err) {
				return errors.Wrapf(err, "fetching referenced %s pvc", pvcName)
			}
			pvc = nil
		}

		// The reconciliation loop should fail if a PVC should be reconciled
		// which is not owned by the operator.
		if pvc != nil && !metav1.IsControlledBy(pvc, r.centralObj) {
			if pvcConfig.StorageClassName != nil ||
				pointer.StringDeref(pvcConfig.Size, "") != "" {
				err := errors.Errorf("Failed reconciling PVC %q. Please remove "+
					"the storageClassName and size properties from your spec, "+
					"or change the name to allow the operator to create a new "+
					"one with a different name.", pvcName)
				r.log.Error(err, "failed reconciling PVC")
				return err
			}
			return nil
		}

		if pvc == nil {
			if err := r.handleCreate(pvcName, pvcConfig); err != nil {
				return err
			}
		}

		if err := r.handleReconcile(pvc, pvcConfig); err != nil {
			return err
		}
	}

	return nil
}

func (r *reconcilePVCExtensionRun) handleDelete() error {
	ownedPVCs, err := r.getOwnedPVCsForCurrentTarget()
	if err != nil {
		return errors.Wrap(err, "fetching operator owned PVCs")
	}

	for _, ownedPVC := range ownedPVCs {
		utils.RemoveOwnerRef(ownedPVC, r.centralObj)
		r.log.Info(fmt.Sprintf("removed owner reference from %q", ownedPVC.GetName()))

		if err := r.client.Update(r.ctx, ownedPVC); err != nil {
			return errors.Wrapf(err, "removing OwnerReference from %s pvc", ownedPVC.GetName())
		}
	}
	return nil
}

func (r *reconcilePVCExtensionRun) handleCreate(claimName string, pvcConfig *platform.PersistentVolumeClaim) error {
	size, err := parseResourceQuantityOr(pvcConfig.Size, defaultPVCSize)
	if err != nil {
		return errors.Wrap(err, "invalid PVC size")
	}
	newPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: r.namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.centralObj, r.centralObj.GroupVersionKind()),
			},
			Labels: map[string]string{
				pvcTargetLabelKey: string(r.target),
			},
		},

		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: size,
				},
			},
			StorageClassName: pvcConfig.StorageClassName,
		},
	}

	if err := r.client.Create(r.ctx, newPVC); err != nil {
		return errors.Wrapf(err, "creating new %s pvc", claimName)
	}
	return nil
}

func (r *reconcilePVCExtensionRun) handleReconcile(existingPVC *corev1.PersistentVolumeClaim, pvcConfig *platform.PersistentVolumeClaim) error {
	shouldUpdate := false

	if pvcSize := pointer.StringDeref(pvcConfig.Size, ""); pvcSize != "" {
		quantity, err := resource.ParseQuantity(pvcSize)
		if err != nil {
			return errors.Wrapf(err, "invalid PVC size %q", pvcSize)
		}
		existingPVC.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: quantity,
		}
		shouldUpdate = true
	}

	if pointer.StringDeref(pvcConfig.StorageClassName, "") != "" && pvcConfig.StorageClassName != existingPVC.Spec.StorageClassName {
		existingPVC.Spec.StorageClassName = pvcConfig.StorageClassName
		shouldUpdate = true
	}

	if shouldUpdate {
		if err := r.client.Update(r.ctx, existingPVC); err != nil {
			return errors.Wrapf(err, "updating %s pvc", existingPVC.GetName())
		}
	}
	return nil
}

func parseResourceQuantityOr(qStrPtr *string, d resource.Quantity) (resource.Quantity, error) {
	qStr := pointer.StringDeref(qStrPtr, "")
	if qStr == "" {
		return d, nil
	}
	q, err := resource.ParseQuantity(qStr)
	if err != nil {
		return resource.Quantity{}, errors.Wrapf(err, "%q", qStr)
	}
	return q, nil
}

func (r *reconcilePVCExtensionRun) getOwnedPVCsForCurrentTarget() ([]*corev1.PersistentVolumeClaim, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}

	if err := r.client.List(r.ctx, pvcList, ctrlClient.InNamespace(r.namespace)); err != nil {
		return nil, errors.Wrapf(err, "receiving list PVC list for %s %s", r.centralObj.GroupVersionKind(), r.centralObj.GetName())
	}

	var pvcs []*corev1.PersistentVolumeClaim
	for i := range pvcList.Items {
		pvc := pvcList.Items[i]
		if r.getTargetLabelValue(&pvc) == string(r.target) {
			pvcs = append(pvcs, &pvc)
		}
	}

	return pvcs, nil
}

func (r *reconcilePVCExtensionRun) getTargetLabelValue(pvc *corev1.PersistentVolumeClaim) string {
	// If the PCV is not owned by the Central we're reconciling, do not make any assumptions about the target.
	if !metav1.IsControlledBy(pvc, r.centralObj) {
		return ""
	}
	if val, ok := pvc.Labels[pvcTargetLabelKey]; ok {
		return val
	}
	// If the target annotation is not set, assume this (owned) PVC is a Central PVC for backwards compatibility.
	return string(PVCTargetCentral)
}
