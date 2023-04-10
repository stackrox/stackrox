package fake

import (
	"context"
	"math/rand"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type networkPolicyToBeManaged struct {
	workload      NetworkPolicyWorkload
	networkPolicy *networkingV1.NetworkPolicy
}

func (w *WorkloadManager) getNetworkPolicy(workload NetworkPolicyWorkload, id string) *networkPolicyToBeManaged {
	namespace := namespacesWithDeploymentsPool.mustGetRandomElem()
	labels := labelsPool.randomElem(namespace)
	np := &networkingV1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      randString(),
			Namespace: namespace,
			UID:       idOrNewUID(id),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			Labels:      labels,
			Annotations: createRandMap(16, 3),
		},
		Spec: networkingV1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Ingress: []networkingV1.NetworkPolicyIngressRule{},
			Egress:  []networkingV1.NetworkPolicyEgressRule{},
			PolicyTypes: []networkingV1.PolicyType{
				networkingV1.PolicyTypeIngress,
				networkingV1.PolicyTypeEgress,
			},
		},
	}
	return &networkPolicyToBeManaged{
		workload:      workload,
		networkPolicy: np,
	}
}

func (w *WorkloadManager) manageNetworkPolicy(ctx context.Context, resources *networkPolicyToBeManaged) {
	w.manageNetworkPolicyLifecycle(ctx, resources)

	for count := 0; resources.workload.NumLifecycles == 0 || count < resources.workload.NumLifecycles; count++ {
		resources = w.getNetworkPolicy(resources.workload, "")
		if _, err := w.client.Kubernetes().NetworkingV1().NetworkPolicies(resources.networkPolicy.Namespace).Create(ctx, resources.networkPolicy, metav1.CreateOptions{}); err != nil {
			log.Errorf("error creating networkPolicy: %v", err)
		}
		w.writeID(networkPolicyPrefix, resources.networkPolicy.UID)
		w.manageNetworkPolicyLifecycle(ctx, resources)
	}
}

func (w *WorkloadManager) manageNetworkPolicyLifecycle(ctx context.Context, resources *networkPolicyToBeManaged) {
	timer := newTimerWithJitter(resources.workload.LifecycleDuration/2 + time.Duration(rand.Int63n(int64(resources.workload.LifecycleDuration))))
	defer timer.Stop()

	npNextUpdate := calculateDurationWithJitter(resources.workload.UpdateInterval)

	np := resources.networkPolicy
	stopSig := concurrency.NewSignal()
	npClient := w.client.Kubernetes().NetworkingV1().NetworkPolicies(np.Namespace)

	for {
		select {
		case <-timer.C:
			stopSig.Signal()
			if err := npClient.Delete(ctx, np.Name, metav1.DeleteOptions{}); err != nil {
				log.Error(err)
			}
			w.deleteID(networkPolicyPrefix, np.UID)
			return
		case <-time.After(npNextUpdate):
			npNextUpdate = calculateDurationWithJitter(resources.workload.UpdateInterval)

			annotations := createRandMap(16, 3)
			np.Annotations = annotations

			if _, err := npClient.Update(ctx, np, metav1.UpdateOptions{}); err != nil {
				log.Error(err)
			}
		}
	}
}
