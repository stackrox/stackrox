package utils

import (
	"context"
	"fmt"

	storagev1 "k8s.io/api/storage/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultStorageClassAnnotationKey is the annotation used to identify a default storage class.
	DefaultStorageClassAnnotationKey = "storageclass.kubernetes.io/is-default-class"
)

// HasDefaultStorageClass tells whether there is a StorageClass marked as a
// default one. Return false if an error occurs when talking to K8s API.
func HasDefaultStorageClass(ctx context.Context, client ctrlClient.Reader) (bool, error) {
	storageClassList := storagev1.StorageClassList{}
	if err := client.List(ctx, &storageClassList); err != nil {
		return false, fmt.Errorf("listing available StorageClasses: %w", err)
	}

	for _, sc := range storageClassList.Items {
		value, hasAnnotation := sc.GetAnnotations()[DefaultStorageClassAnnotationKey]
		if hasAnnotation && value == "true" {
			return true, nil
		}
	}

	return false, nil
}
