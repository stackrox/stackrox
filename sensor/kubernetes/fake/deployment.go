package fake

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/containerid"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/sync"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	processPool = newProcessPool()
)

// ProcessPool stores processes by containerID using a map
type ProcessPool struct {
	Processes map[string][]*storage.ProcessSignal
	Capacity  int
	Size      int
	lock      sync.RWMutex
}

func newProcessPool() *ProcessPool {
	return &ProcessPool{
		Processes: make(map[string][]*storage.ProcessSignal),
		Capacity:  10000,
		Size:      0,
	}
}

func (p *ProcessPool) add(val *storage.ProcessSignal) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.Size < p.Capacity {
		p.Processes[val.ContainerId] = append(p.Processes[val.ContainerId], val)
		p.Size++
	} else {
		nprocess := len(p.Processes[val.ContainerId])
		if nprocess > 0 {
			randIdx := rand.Intn(nprocess)
			p.Processes[val.ContainerId][randIdx] = val
		}
	}
}

func (p *ProcessPool) remove(containerID string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.Size -= len(p.Processes[containerID])
	delete(p.Processes, containerID)
}

func (p *ProcessPool) getRandomProcess(containerID string) *storage.ProcessSignal {
	p.lock.Lock()
	defer p.lock.Unlock()

	size := len(p.Processes[containerID])
	if size > 0 {
		randIdx := rand.Intn(size)
		return p.Processes[containerID][randIdx]
	}

	return nil
}

type deploymentResourcesToBeManaged struct {
	workload DeploymentWorkload

	deployment *appsv1.Deployment
	replicaSet *appsv1.ReplicaSet
	pods       []*corev1.Pod
}

func createRandMap(stringSize, entries int) map[string]string {
	m := make(map[string]string, entries)
	for i := 0; i < entries; i++ {
		m[randStringWithLength(stringSize)] = randStringWithLength(stringSize)
	}
	return m
}

func createMap(entries int) map[string]string {
	m := make(map[string]string, entries)
	for i := 0; i < entries; i++ {
		m[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}
	return m
}

func createDeploymentLabels(random bool, numLabels int) map[string]string {
	if random {
		return createRandMap(16, numLabels)
	}
	return createMap(numLabels)
}

func (w *WorkloadManager) getDeployment(workload DeploymentWorkload, idx int, deploymentIDs, replicaSetIDs, podIDs []string) *deploymentResourcesToBeManaged {
	var labels map[string]string
	if workload.NumLabels == 0 {
		labels = createDeploymentLabels(workload.RandomLabels, 3)
	} else {
		labels = createDeploymentLabels(workload.RandomLabels, workload.NumLabels)
	}

	var containers []corev1.Container
	for i := 0; i < workload.PodWorkload.NumContainers; i++ {
		containers = append(containers, getContainer(workload.PodWorkload.ContainerWorkload))
	}

	namespace, valid := namespacePool.randomElem()
	if !valid {
		namespace = "default"
	}

	labelsPool.add(namespace, labels)
	namespacesWithDeploymentsPool.add(namespace)

	var serviceAccount string
	potentialServiceAccounts := serviceAccountPool[namespace]
	if len(potentialServiceAccounts) == 0 {
		serviceAccount = "default"
	} else {
		serviceAccount = potentialServiceAccounts[rand.Intn(len(potentialServiceAccounts))]
	}
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      randString(),
			Namespace: namespace,
			UID:       idOrNewUID(getID(deploymentIDs, idx)),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			Labels:      labels,
			Annotations: createRandMap(16, 3),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointers.Int32(int32(workload.PodWorkload.NumPods)),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   namespace,
					Labels:      labels,
					Annotations: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "vol1",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/host",
								},
							},
						},
						{
							Name: "vol2",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: "RAM",
								},
							},
						},
						{
							Name: "vol3",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "secret",
								},
							},
						},
						{
							Name: "vol4",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "stackrox-db",
								},
							},
						},
					},
					Containers:                   containers,
					AutomountServiceAccountToken: pointers.Bool(true),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: pointers.Bool(true),
					},
					ServiceAccountName: serviceAccount,
				},
			},
		},
	}
	w.writeID(deploymentPrefix, deployment.UID)

	rs := getReplicaSet(deployment, getID(replicaSetIDs, idx))
	w.writeID(replicaSetPrefix, rs.UID)

	var pods []*corev1.Pod
	for i := 0; i < workload.PodWorkload.NumPods; i++ {
		pod := getPod(rs, getID(podIDs, i+idx*workload.PodWorkload.NumPods))
		w.writeID(podPrefix, pod.UID)
		pods = append(pods, pod)
	}
	return &deploymentResourcesToBeManaged{
		workload:   workload,
		deployment: deployment,
		replicaSet: rs,
		pods:       pods,
	}
}

func getReplicaSet(deployment *appsv1.Deployment, id string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       kubernetes.ReplicaSet,
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      randString(),
			Namespace: deployment.Namespace,
			UID:       idOrNewUID(id),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			Labels:      deployment.Labels,
			Annotations: deployment.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       kubernetes.Deployment,
					Name:       deployment.Name,
					UID:        deployment.UID,
				},
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: deployment.Spec.Replicas,
			Selector: deployment.Spec.Selector,
			Template: corev1.PodTemplateSpec{
				Spec: deployment.Spec.Template.Spec,
			},
		},
	}
}

func getPod(replicaSet *appsv1.ReplicaSet, id string) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      randString(),
			Namespace: replicaSet.Namespace,
			UID:       idOrNewUID(id),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			Labels:      replicaSet.Labels,
			Annotations: createRandMap(16, 3),
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: kubernetes.ReplicaSet,
					UID:  replicaSet.UID,
				},
			},
		},
		Spec: replicaSet.Spec.Template.Spec,
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			StartTime: &metav1.Time{
				Time: time.Now(),
			},
			PodIP: generateAndAddIPToPool(),
		},
	}
	populatePodContainerStatuses(pod)
	return pod
}

func getContainer(workload ContainerWorkload) corev1.Container {
	var imageName string
	if workload.NumImages == 0 {
		imageName = fixtures.GetRandomImage().FullName()
	} else {
		imageName = fixtures.GetRandomImageN(workload.NumImages).FullName()
	}
	return corev1.Container{
		Name:  randString(),
		Image: imageName,
		Command: []string{
			"sleep",
			"6000",
		},
		Args: []string{
			"more",
			"sleep",
		},
		Ports: []corev1.ContainerPort{
			{
				Name:     "http-port",
				HostPort: 8080,
				Protocol: "TCP",
			},
			{
				Name:          "https-port",
				ContainerPort: 443,
				Protocol:      "TCP",
			},
			{
				Name:          "tcp-port",
				ContainerPort: 8443,
				Protocol:      "TCP",
			},
			{
				Name:          "api",
				ContainerPort: 8081,
				Protocol:      "TCP",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "ROX_FEATURE_FLAG",
				Value: "true",
			},
			{
				Name:  "ROX_TOKEN",
				Value: "toxtoken",
			},
			{
				Name:  "ROX_API_TOKEN",
				Value: "roxapitoken",
			},
			{
				Name:  "ROX_SECRET_PASSWORD",
				Value: "secretpassword",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "rox-secret",
						},
						Key: "db-password",
					},
				},
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("1G"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("1G"),
			},
		},
		VolumeMounts:    nil,
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{},
	}
}

// The jitter added is a random amount between -1 to 1 second
func calculateDurationWithJitter(duration time.Duration) time.Duration {
	jitter := rand.Int63n(int64(2*time.Second)) - int64(1*time.Second)
	return duration + time.Duration(jitter)
}

func newTimerWithJitter(duration time.Duration) *time.Timer {
	return time.NewTimer(calculateDurationWithJitter(duration))
}

// manageDeployment takes in the initial resources and then will recreate them when they are deleted
// this function should be called with go w.manageDeployment
func (w *WorkloadManager) manageDeployment(ctx context.Context, resources *deploymentResourcesToBeManaged) {
	// Handle resources that were initialized for initial startup. These start up resources
	// are like deploying Sensor into a new environment and syncing all objects
	w.manageDeploymentLifecycle(ctx, resources)

	// The previous function returning means that the deployments, replicaset and pods were all deleted
	// Now we recreate the objects again
	for count := 0; resources.workload.NumLifecycles == 0 || count < resources.workload.NumLifecycles; count++ {
		resources = w.getDeployment(resources.workload, 0, nil, nil, nil)
		deployment, replicaSet, pods := resources.deployment, resources.replicaSet, resources.pods
		if _, err := w.client.Kubernetes().AppsV1().Deployments(deployment.Namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
			log.Errorf("error creating deployment: %v", err)
		}
		if _, err := w.client.Kubernetes().AppsV1().ReplicaSets(deployment.Namespace).Create(ctx, replicaSet, metav1.CreateOptions{}); err != nil {
			log.Errorf("error creating replica set: %v", err)
		}
		for _, pod := range pods {
			if _, err := w.client.Kubernetes().CoreV1().Pods(deployment.Namespace).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
				log.Errorf("error creating pod: %v", err)
			}
		}
		w.manageDeploymentLifecycle(ctx, resources)
	}
}

func (w *WorkloadManager) manageDeploymentLifecycle(ctx context.Context, resources *deploymentResourcesToBeManaged) {
	timer := newTimerWithJitter(resources.workload.LifecycleDuration/2 + time.Duration(rand.Int63n(int64(resources.workload.LifecycleDuration))))
	defer timer.Stop()

	deploymentNextUpdate := calculateDurationWithJitter(resources.workload.UpdateInterval)

	deployment := resources.deployment
	replicaset := resources.replicaSet

	stopSig := concurrency.NewSignal()
	deploymentClient := w.client.Kubernetes().AppsV1().Deployments(deployment.Namespace)
	replicaSetClient := w.client.Kubernetes().AppsV1().ReplicaSets(deployment.Namespace)

	for _, pod := range resources.pods {
		go w.managePod(ctx, &stopSig, resources.workload.PodWorkload, pod)
	}

	for {
		select {
		case <-timer.C:
			stopSig.Signal()
			if err := deploymentClient.Delete(ctx, deployment.Name, metav1.DeleteOptions{}); err != nil {
				log.Error(err)
			}
			w.deleteID(deploymentPrefix, deployment.UID)
			if err := replicaSetClient.Delete(ctx, replicaset.Name, metav1.DeleteOptions{}); err != nil {
				log.Error(err)
			}
			w.deleteID(replicaSetPrefix, replicaset.UID)
			return
		case <-time.After(deploymentNextUpdate):
			deploymentNextUpdate = calculateDurationWithJitter(resources.workload.UpdateInterval)

			annotations := createRandMap(16, 3)

			deployment.Annotations = annotations
			replicaset.Annotations = annotations

			if _, err := deploymentClient.Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
				log.Errorf("error updating deployment: %v", err)
			}
			if _, err := replicaSetClient.Update(ctx, replicaset, metav1.UpdateOptions{}); err != nil {
				log.Errorf("error updating replica set: %v", err)
			}
		}
	}
}

func populatePodContainerStatuses(pod *corev1.Pod) {
	statuses := make([]corev1.ContainerStatus, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		status := corev1.ContainerStatus{
			Name:        container.Name,
			State:       corev1.ContainerState{},
			Ready:       true,
			Image:       container.Image,
			ImageID:     fmt.Sprintf("docker-pullable://%s", container.Image),
			ContainerID: fmt.Sprintf("docker://%s", randStringWithLength(63)),
		}
		containerPool.add(getShortContainerID(status.ContainerID))
		statuses = append(statuses, status)
	}
	pod.Status.ContainerStatuses = statuses
}

func (w *WorkloadManager) managePod(ctx context.Context, deploymentSig *concurrency.Signal, podWorkload PodWorkload, pod *corev1.Pod) {
	podDeadline := newTimerWithJitter(podWorkload.LifecycleDuration)
	defer podDeadline.Stop()

	podSig := concurrency.NewSignal()
	go w.manageProcessesForPod(&podSig, podWorkload, pod)

	client := w.client.Kubernetes().CoreV1().Pods(pod.Namespace)
	cleanupPodFn := func(pod *corev1.Pod) {
		if err := client.Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
			log.Errorf("error deleting pod: %v", err)
		}
		w.deleteID(podPrefix, pod.UID)
		ipPool.remove(pod.Status.PodIP)

		for _, cs := range pod.Status.ContainerStatuses {
			containerPool.remove(getShortContainerID(cs.ContainerID))
		}
		podSig.Signal()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-deploymentSig.Done():
			// Deployment has been deleted so delete pod
			cleanupPodFn(pod)
			return
		case <-podDeadline.C:
			cleanupPodFn(pod)

			// New pod name and UUID
			pod.Name = randString()
			pod.UID = newUUID()
			pod.Status.PodIP = generateAndAddIPToPool()
			populatePodContainerStatuses(pod)

			if _, err := client.Create(ctx, pod, metav1.CreateOptions{}); err != nil {
				log.Errorf("error creating pod: %v", err)
			}
			w.writeID(podPrefix, pod.UID)
			podSig = concurrency.NewSignal()
			go w.manageProcessesForPod(&podSig, podWorkload, pod)
			podDeadline = newTimerWithJitter(podWorkload.LifecycleDuration)
		}
	}
}

func getShortContainerID(id string) string {
	_, runtimeID := k8sutil.ParseContainerRuntimeString(id)
	return containerid.ShortContainerIDFromInstanceID(runtimeID)
}

func (w *WorkloadManager) manageProcessesForPod(podSig *concurrency.Signal, podWorkload PodWorkload, pod *corev1.Pod) {
	processWorkload := podWorkload.ProcessWorkload

	if processWorkload.ProcessInterval == 0 {
		return
	}
	ticker := time.NewTicker(processWorkload.ProcessInterval)
	defer ticker.Stop()

	// Precompute these as multiple calls to getShortContainerID is expensive
	containerIDs := make([]string, 0, len(pod.Status.ContainerStatuses))
	for _, status := range pod.Status.ContainerStatuses {
		containerIDs = append(containerIDs, getShortContainerID(status.ContainerID))
	}
	for {
		select {
		case <-ticker.C:
			if !w.servicesInitialized.IsDone() {
				continue
			}

			containerID := containerIDs[rand.Intn(len(containerIDs))]

			if processWorkload.ActiveProcesses {
				for _, process := range getActiveProcesses(containerID) {
					w.processes.Process(process)
					processPool.add(process)
				}
			} else {
				// If less than the rate, then it's a bad process
				if rand.Float32() < processWorkload.AlertRate {
					w.processes.Process(getBadProcess(containerID))
				} else {
					goodProcess := getGoodProcess(containerID)
					w.processes.Process(goodProcess)
					processPool.add(goodProcess)
				}
			}
		case <-podSig.Done():
			return
		}
	}
}
