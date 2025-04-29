package extensions

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	utils "github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/pkg/sliceutils"
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

	// DefaultCentralDBBackupPVCName is the default name for Central DB backup PVC
	DefaultCentralDBBackupPVCName = "central-db-backup"

	pvcTargetLabelKey = "target.pvc.stackrox.io"
)

// PVCTarget specifies which deployment should attach the PVC
type PVCTarget string

const (
	// PVCTargetCentral is for any PVC that would be attached to the Central deployment
	PVCTargetCentral PVCTarget = "central"

	// PVCTargetCentralDB is for a PVC that would be attached to the Central DB
	// deployment as a data volume
	PVCTargetCentralDB PVCTarget = "central-db"

	// PVCTargetCentralDBBackup is for a PVC that would be attached to the
	// Central DB deployment as a backup volume
	PVCTargetCentralDBBackup PVCTarget = "central-db-backup"
)

var (
	errMultipleOwnedPVCs = errors.New("operator is only allowed to have 1 owned PVC")

	DefaultPVCSize       = resource.MustParse("100Gi")
	DefaultBackupPVCSize = resource.MustParse("200Gi") // 2*DefaultPVCSize
)

func convertDBPersistenceToPersistence(p *platform.DBPersistence, target PVCTarget) (*platform.Persistence, error) {
	if p == nil {
		return nil, nil
	}
	if p.HostPath != nil {
		return &platform.Persistence{
			HostPath: p.HostPath,
		}, nil
	}
	pvc := p.GetPersistentVolumeClaim()
	if pvc == nil {
		return &platform.Persistence{}, nil
	}

	claimName := pvc.ClaimName
	pvcSize := pvc.Size

	if target == PVCTargetCentralDBBackup {
		if claimName != nil {
			// If a ClaimName is specified, derive the backup PVC ClamName from
			// it as well. We don't want to modify the pointer in place, make a
			// copy instead -- otherwise the next reconciliation will repeat the
			// modification, duplicating the suffix.
			backupName := fmt.Sprintf("%s-backup", *claimName)
			claimName = &backupName
		}

		if pvcSize != nil {
			// If a Size is specified, derive the backup PVC Size from it as
			// well, the rule of thumb is that it should be twice as large, to
			// accomodate the backup and one restore copy.
			//
			// The same as above, we don't want to modify the pointer in place,
			// make a copy instead -- otherwise the next reconciliation will
			// repeat the modification, duplicating the suffix.
			quantity, err := resource.ParseQuantity(*pvcSize)

			if err != nil {
				return nil, errors.Wrap(err, "failed to calculate backup volume size")
			} else {
				quantity.Mul(2)
				backupSize := quantity.String()
				pvcSize = &backupSize
			}
		}
	}

	return &platform.Persistence{
		PersistentVolumeClaim: &platform.PersistentVolumeClaim{
			ClaimName:        claimName,
			Size:             pvcSize,
			StorageClassName: pvc.StorageClassName,
		},
	}, nil
}

// getPersistenceByTarget retrieves the persistence configuration for the given
// PVC target:
//   - PVCTargetCentral -- the embedded persistent volume on which RocksDB is
//     stored
//   - PVCTargetCentralDB -- the persistent volume that serves as the backing
//     store for the central-db PostgreSQL database
//   - PVCTargetCentralDBBackup -- the persistent volume for storing PostgreSQL
//     database backups
//
// A nil return value indicates that no persistent volume should be provisioned for the respective target.
func getPersistenceByTarget(central *platform.Central, target PVCTarget, log logr.Logger) (*platform.Persistence, error) {
	switch target {
	case PVCTargetCentral:
		return nil, nil
	case PVCTargetCentralDB, PVCTargetCentralDBBackup:
		if !central.Spec.Central.ShouldManageDB() {
			return nil, nil
		}
		dbPersistence := central.Spec.Central.GetDB().GetPersistence()
		if dbPersistence == nil {
			dbPersistence = &platform.DBPersistence{}
		}

		return convertDBPersistenceToPersistence(dbPersistence, target, log)
	default:
		return nil, errors.Errorf("unknown pvc target %q", target)
	}
}

func getDefaultPVCSizeByTarget(target PVCTarget) (resource.Quantity, error) {
	switch target {
	case PVCTargetCentral:
		return resource.MustParse("0"), nil
	case PVCTargetCentralDB:
		return DefaultPVCSize, nil
	case PVCTargetCentralDBBackup:
		return DefaultBackupPVCSize, nil
	default:
		return resource.MustParse("0"), errors.Errorf("unknown pvc target %q", target)
	}
}

// ReconcilePVCExtension reconciles PVCs created by the operator. The PVC is not managed by a Helm chart
// because if a user uninstalls StackRox, it should keep the data, preventing to unintentionally erasing data.
// On uninstall the owner reference is removed from the PVC objects.
func ReconcilePVCExtension(client ctrlClient.Client, direct ctrlClient.Reader, target PVCTarget, defaultClaimName string, opts ...PVCOption) extensions.ReconcileExtension {

	fn := func(ctx context.Context, central *platform.Central, client ctrlClient.Client, direct ctrlClient.Reader, _ func(statusFunc updateStatusFunc), log logr.Logger) error {
		persistence, err := getPersistenceByTarget(central, target, log)
		if err != nil {
			return err
		}

		return reconcilePVC(ctx, central, persistence, target, defaultClaimName, client, log, opts...)
	}
	return wrapExtension(fn, client, direct)
}

func reconcilePVC(ctx context.Context, central *platform.Central, persistence *platform.Persistence, target PVCTarget, defaultClaimName string, client ctrlClient.Client, log logr.Logger, opts ...PVCOption) error {
	defaultPVCSize, err := getDefaultPVCSizeByTarget(target)
	if err != nil {
		return errors.Wrapf(err, "Could not get default PVC size")
	}

	ext := reconcilePVCExtensionRun{
		ctx:              ctx,
		namespace:        central.GetNamespace(),
		client:           client,
		centralObj:       central,
		persistence:      persistence,
		target:           target,
		defaultClaimName: defaultClaimName,
		defaultClaimSize: defaultPVCSize,
		log:              log.WithValues("pvcReconciler", target),
	}

	for _, option := range opts {
		option(&ext)
	}

	return ext.Execute()
}

type reconcilePVCExtensionRun struct {
	ctx              context.Context
	namespace        string
	client           ctrlClient.Client
	centralObj       *platform.Central
	persistence      *platform.Persistence
	defaultClaimSize resource.Quantity
	defaultClaimName string
	target           PVCTarget
	log              logr.Logger
}

type PVCOption func(*reconcilePVCExtensionRun)

func WithDefaultClaimSize(value resource.Quantity) PVCOption {
	return func(r *reconcilePVCExtensionRun) {
		r.defaultClaimSize = value
	}
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
	key := ctrlClient.ObjectKey{Namespace: r.namespace, Name: claimName}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.client.Get(r.ctx, key, pvc); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "fetching referenced %s pvc", claimName)
		}
		pvc = nil
	}

	ownedPVC, err := r.getUniqueOwnedPVCForCurrentTarget()
	if err != nil {
		return err
	}
	if ownedPVC != nil {
		if ownedPVC.GetName() != claimName && pvc == nil {
			return errors.Errorf(
				"Could not create PVC %q because the operator can only manage 1 PVC for %s. To fix this either reference a manually created PVC or remove the OwnerReference of the %q PVC.", claimName, r.target, ownedPVC.GetName())
		}
	}

	// The reconciliation loop should fail if a PVC should be reconciled which is not owned by the operator.
	if pvc != nil && !metav1.IsControlledBy(pvc, r.centralObj) {
		if pvcConfig.StorageClassName != nil || pointer.StringDeref(pvcConfig.Size, "") != "" {
			err := errors.Errorf("Failed reconciling PVC %q. Please remove the storageClassName and size properties from your spec, or change the name to allow the operator to create a new one with a different name.", claimName)
			r.log.Error(err, "failed reconciling PVC")
			return err
		}
		return nil
	}

	if pvc == nil {
		return r.handleCreate(claimName, pvcConfig)
	}

	return r.handleReconcile(pvc, pvcConfig)
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
	// Before creating a PVC, verify if prerequisites are met. Currently there
	// is only one requirement, a default storage class must exists or a
	// storage class has to be specified explicitly. Since it's highly specific
	// for PVCs only, it's implemented inside the extension, instead of
	// collecting this information at the start and passing it into the
	// extension.
	//
	// Note that to make this check less disruptive, in case if we face an
	// error we still try to create a PVC.
	hasDefault, err := utils.HasDefaultStorageClass(r.ctx, r.client)
	if err != nil {
		r.log.Error(err, fmt.Sprintf("cannot find the default storage class, but proceeding with %q PVC creation", claimName))
	} else {
		if !hasDefault && pvcConfig.StorageClassName == nil {
			// For the backup PVC it's a hard stop
			if r.target == PVCTargetCentralDBBackup {
				r.log.Info(fmt.Sprintf("No default storage class or explicit storage class found, skip %q PVC creation", claimName))
				return nil
			} else {
				r.log.Info(fmt.Sprintf("No default storage class or explicit storage class found, but proceeding with %q PVC creation", claimName))
			}
		}
	}

	size, err := parseResourceQuantityOr(pvcConfig.Size, r.defaultClaimSize)
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

func (r *reconcilePVCExtensionRun) getUniqueOwnedPVCForCurrentTarget() (*corev1.PersistentVolumeClaim, error) {
	pvcList, err := r.getOwnedPVCsForCurrentTarget()
	if err != nil {
		return nil, err
	}

	// If no previously created managed PVC was found everything is ok.
	if len(pvcList) == 0 {
		return nil, nil
	}
	if len(pvcList) > 1 {
		names := sliceutils.Map(pvcList, (*corev1.PersistentVolumeClaim).GetName)
		slices.Sort(names)

		return nil, errors.Wrapf(errMultipleOwnedPVCs,
			"multiple owned PVCs were found for %s, please remove not used ones or delete their OwnerReferences. Found PVCs: %s", r.target, strings.Join(names, ", "))
	}

	return pvcList[0], nil
}
