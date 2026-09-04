package complianceoperator

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const defaultStorageClassAnnotationKey = "storageclass.kubernetes.io/is-default-class"

var storageClassGVR = schema.GroupVersionResource{
	Group:    "storage.k8s.io",
	Version:  "v1",
	Resource: "storageclasses",
}

// Customer-facing message aligned with Red Hat / OpenShift wording:
// - Prefer "storage class" for the concept; StorageClass is the API object.
// - Do not mention ScanSetting: RHACS creates that CR in openshift-compliance;
//   customers creating schedules in the RHACS UI cannot set storageClassName there.
const noDefaultStorageClassErrMsg = "No default storage class is configured on the cluster. Compliance scans require a default storage class so that persistent volumes can be provisioned for raw scan results. Set a storage class as the default before creating a scan schedule. For example, run: oc annotate storageclass <storage_class_name> storageclass.kubernetes.io/is-default-class=true"

func hasDefaultStorageClass(ctx context.Context, client dynamic.Interface) (bool, error) {
	list, err := client.Resource(storageClassGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, errors.Wrap(err, "listing StorageClasses")
	}

	for _, item := range list.Items {
		if item.GetAnnotations()[defaultStorageClassAnnotationKey] == "true" {
			return true, nil
		}
	}

	return false, nil
}

func (m *handlerImpl) validateComplianceScanStorage(ctx context.Context) error {
	hasDefault, err := hasDefaultStorageClass(ctx, m.client)
	if err != nil {
		// Fail closed: if we cannot determine whether a default storage class exists,
		// do not create scan resources that may leave PVCs Pending indefinitely.
		return errors.Wrap(err, "unable to verify default storage class")
	}
	if !hasDefault {
		return errors.New(noDefaultStorageClassErrMsg)
	}
	return nil
}
