package backgroundmigrations

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const deploymentName = "central"

var log = logging.LoggerForModule()

// RolloutChecker checks whether the Central deployment rollout is complete.
type RolloutChecker interface {
	IsRolloutDone(ctx context.Context) (bool, error)
}

type k8sRolloutChecker struct {
	client    kubernetes.Interface
	inCluster bool
}

// NewCentralRolloutChecker creates a RolloutChecker that queries the K8s API.
func NewCentralRolloutChecker() RolloutChecker {
	cfg, err := rest.InClusterConfig()
	rc := &k8sRolloutChecker{inCluster: true}

	if err != nil {
		log.Warnf("failed to get in cluster kubernetes config, assuming not running in a kubernetes cluster: %v", err)
		rc.inCluster = false
		return rc
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Warnf("failed to create in cluster K8s client, assuming not running in a kubernetes cluster: %v", err)
		rc.inCluster = false
		return rc
	}

	rc.client = client

	return rc
}

// IsRolloutDone checks whether the Central deployment rollout is complete.
func (c *k8sRolloutChecker) IsRolloutDone(ctx context.Context) (bool, error) {
	if !c.inCluster {
		return true, nil
	}

	namespace := env.Namespace.Setting()
	deployment, err := c.client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("error getting deployment %s/%s: %w", namespace, deploymentName, err)
	}

	if done := c.checkRolloutStatus(deployment); !done {
		return false, nil
	}

	terminatingPods, err := c.checkTerminatingPods(ctx, deployment)
	if err != nil {
		return false, err
	}

	if len(terminatingPods) > 0 {
		log.Infof("deployment %s/%s rollout complete but pods still terminating: %v",
			namespace, deploymentName, terminatingPods)
		return false, nil
	}

	log.Infof("deployment %s/%s rollout complete, no terminating pods", namespace, deploymentName)
	return true, nil
}

func (c *k8sRolloutChecker) checkRolloutStatus(deployment *appsv1.Deployment) bool {
	replicas := int32(1)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}
	if deployment.Status.UpdatedReplicas != replicas ||
		deployment.Status.AvailableReplicas != replicas ||
		deployment.Status.ObservedGeneration < deployment.Generation {
		namespace := env.Namespace.Setting()
		log.Infof("deployment %s/%s rollout in progress (updated=%d, available=%d, desired=%d)",
			namespace, deploymentName,
			deployment.Status.UpdatedReplicas,
			deployment.Status.AvailableReplicas,
			replicas)
		return false
	}
	return true
}

// checkTerminatingPods queries for terminating pods matching the deployment selector
// TODO: once the min. k8s version we support is 1.35 we can use deployment.Status.TerminatingReplicas instead
func (c *k8sRolloutChecker) checkTerminatingPods(ctx context.Context, deployment *appsv1.Deployment) ([]string, error) {
	namespace := env.Namespace.Setting()

	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("error parsing deployment selector: %w", err)
	}

	pods, err := c.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("error listing pods for deployment %s/%s: %w", namespace, deploymentName, err)
	}

	var terminating []string
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			terminating = append(terminating, pod.Name)
		}
	}
	return terminating, nil
}
