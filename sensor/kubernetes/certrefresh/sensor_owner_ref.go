package certrefresh

import (
	"context"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

// FetchSensorDeploymentOwnerRef retrieves the OwnerReference of the Deployment that controls a specific sensor Pod.
// It follows the owner hierarchy of the pod and its ReplicaSet to return the top-level Deployment reference.
func FetchSensorDeploymentOwnerRef(ctx context.Context, sensorPodName, sensorNamespace string,
	k8sClient kubernetes.Interface, backoff wait.Backoff) (*metav1.OwnerReference, error) {
	if sensorPodName == "" {
		return nil, errors.New("fetching sensor deployment: empty pod name")
	}

	podsClient := k8sClient.CoreV1().Pods(sensorNamespace)
	sensorPodMeta, getPodErr := getObjectMetaWithRetries(ctx, backoff, func(ctx context.Context) (metav1.Object, error) {
		pod, err := podsClient.Get(ctx, sensorPodName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return pod.GetObjectMeta(), nil
	})
	if getPodErr != nil {
		return nil, errors.Wrapf(getPodErr, "fetching sensor pod with name %q", sensorPodName)
	}
	podOwners := sensorPodMeta.GetOwnerReferences()
	if len(podOwners) != 1 {
		return nil, errors.Errorf("pod %q has unexpected owners %v",
			sensorPodName, podOwners)
	}
	podOwnerName := podOwners[0].Name

	replicaSetClient := k8sClient.AppsV1().ReplicaSets(sensorNamespace)
	ownerReplicaSetMeta, getReplicaSetErr := getObjectMetaWithRetries(ctx, backoff,
		func(ctx context.Context) (metav1.Object, error) {
			replicaSet, err := replicaSetClient.Get(ctx, podOwnerName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return replicaSet.GetObjectMeta(), nil
		})
	if getReplicaSetErr != nil {
		return nil, errors.Wrapf(getReplicaSetErr, "fetching owner replica set with name %q", podOwnerName)
	}
	replicaSetOwners := ownerReplicaSetMeta.GetOwnerReferences()
	if len(replicaSetOwners) != 1 {
		return nil, errors.Errorf("replica set %q has unexpected owners %v",
			ownerReplicaSetMeta.GetName(),
			replicaSetOwners)
	}
	replicaSetOwner := replicaSetOwners[0]

	blockOwnerDeletion := false
	isController := false
	return &metav1.OwnerReference{
		APIVersion:         replicaSetOwner.APIVersion,
		Kind:               replicaSetOwner.Kind,
		Name:               replicaSetOwner.Name,
		UID:                replicaSetOwner.UID,
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}, nil
}

func getObjectMetaWithRetries(
	ctx context.Context,
	backoff wait.Backoff,
	getObject func(context.Context) (metav1.Object, error),
) (metav1.Object, error) {
	var object metav1.Object
	getErr := retry.OnError(backoff, func(err error) bool {
		return !k8sErrors.IsNotFound(err)
	}, func() error {
		newObject, err := getObject(ctx)
		if err == nil {
			object = newObject
		}
		return err
	})

	return object, getErr
}
