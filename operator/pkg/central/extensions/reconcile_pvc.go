package extensions

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	utils "github.com/stackrox/rox/operator/pkg/utils"
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

func getPersistenceByClaimName(central *platform.Central, claim string) *platform.Persistence {
	switch claim {
	case DefaultCentralPVCName:
		return central.Spec.Central.GetPersistence()
	case DefaultCentralDBPVCName:
		return convertDBPersistenceToPersistence(central.Spec.Central.DB.GetPersistence())
	default:
		panic("unknown default claim name")
	}
}

// ReconcilePVCExtension reconciles PVCs created by the operator
func ReconcilePVCExtension(client ctrlClient.Client, target PVCTarget, defaultClaimName string) extensions.ReconcileExtension {
	fn := func(ctx context.Context, central *platform.Central, client ctrlClient.Client, _ func(statusFunc updateStatusFunc), log logr.Logger) error {
		persistence := getPersistenceByClaimName(central, defaultClaimName)
		return reconcilePVC(ctx, central, persistence, target, defaultClaimName, client, log)
	}
	return wrapExtension(fn, client)
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
		log:              log,
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
	if r.centralObj.DeletionTimestamp != nil {
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

	claimName := pointer.StringPtrDerefOr(pvcConfig.ClaimName, r.defaultClaimName)
	key := ctrlClient.ObjectKey{Namespace: r.namespace, Name: claimName}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.client.Get(r.ctx, key, pvc); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "fetching referenced %s pvc", claimName)
		}
		pvc = nil
	}

	ownedPVC, err := r.getUniqueOwnedPVCsForCurrentTarget()
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
		if pvcConfig.StorageClassName != nil || pointer.StringPtrDerefOr(pvcConfig.Size, "") != "" {
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
	ownedPVCs, err := r.getOwnedPVC()
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
			Resources: corev1.ResourceRequirements{
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

	if pvcSize := pointer.StringPtrDerefOr(pvcConfig.Size, ""); pvcSize != "" {
		quantity, err := resource.ParseQuantity(pvcSize)
		if err != nil {
			return errors.Wrapf(err, "invalid PVC size %q", pvcSize)
		}
		existingPVC.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: quantity,
		}
		shouldUpdate = true
	}

	if pointer.StringPtrDerefOr(pvcConfig.StorageClassName, "") != "" && pvcConfig.StorageClassName != existingPVC.Spec.StorageClassName {
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
	qStr := pointer.StringPtrDerefOr(qStrPtr, "")
	if qStr == "" {
		return d, nil
	}
	q, err := resource.ParseQuantity(qStr)
	if err != nil {
		return resource.Quantity{}, errors.Wrapf(err, "%q", qStr)
	}
	return q, nil
}

func (r *reconcilePVCExtensionRun) getOwnedPVC() ([]*corev1.PersistentVolumeClaim, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}

	if err := r.client.List(r.ctx, pvcList, ctrlClient.InNamespace(r.namespace)); err != nil {
		return nil, errors.Wrapf(err, "receiving list PVC list for %s %s", r.centralObj.GroupVersionKind(), r.centralObj.GetName())
	}

	var ownedPVCs []*corev1.PersistentVolumeClaim
	for i := range pvcList.Items {
		item := pvcList.Items[i]
		if metav1.IsControlledBy(&item, r.centralObj) {
			tmp := item
			ownedPVCs = append(ownedPVCs, &tmp)
		}
	}

	return ownedPVCs, nil
}

func (r *reconcilePVCExtensionRun) getUniqueOwnedPVCsForCurrentTarget() (*corev1.PersistentVolumeClaim, error) {
	pvcList, err := r.getOwnedPVC()
	if err != nil {
		return nil, err
	}

	// Filter PVC List by current PVC Claim Name
	filtered := make([]*corev1.PersistentVolumeClaim, 0, len(pvcList))
	for _, pvc := range pvcList {
		// If the target annotation is empty, default to Central for backwards compatibility
		if val, ok := pvc.Labels[pvcTargetLabelKey]; !ok && r.target == PVCTargetCentral {
			filtered = append(filtered, pvc)
		} else if val == string(r.target) {
			filtered = append(filtered, pvc)
		}
	}
	// If no previously created managed PVC was found everything is ok.
	if len(filtered) == 0 {
		return nil, nil
	}
	if len(filtered) > 1 {
		var names []string
		for _, item := range filtered {
			names = append(names, item.GetName())
		}
		sort.Strings(names)

		return nil, errors.Wrapf(errMultipleOwnedPVCs,
			"multiple owned PVCs were found for %s, please remove not used ones or delete their OwnerReferences. Found PVCs: %s", r.target, strings.Join(names, ", "))
	}

	return filtered[0], nil
}
